package redisclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidTarget = errors.New("redisclient: invalid target")
	ErrUnavailable   = errors.New("redisclient: unavailable")
)

func Options(target, caFile, serverName string) (*redis.Options, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("redisclient: Redis target is empty: %w", ErrInvalidTarget)
	}

	var options *redis.Options
	var err error
	if strings.Contains(target, "://") {
		options, err = redis.ParseURL(target)
		if err != nil {
			return nil, fmt.Errorf("redisclient: invalid Redis URL for %s: %w", SafeTarget(target), ErrInvalidTarget)
		}
	} else {
		options = &redis.Options{Addr: target}
	}

	options.ReadTimeout = 3 * time.Second
	options.WriteTimeout = 3 * time.Second
	options.PoolTimeout = 4 * time.Second
	options.PoolSize = 20

	tlsRequested := options.TLSConfig != nil
	if !tlsRequested && (strings.TrimSpace(caFile) != "" || strings.TrimSpace(serverName) != "") {
		return nil, fmt.Errorf("redisclient: REDIS_CA_CERT_FILE and REDIS_TLS_SERVER_NAME require rediss://: %w", ErrInvalidTarget)
	}
	if tlsRequested {
		if options.TLSConfig == nil {
			options.TLSConfig = &tls.Config{}
		}
		options.TLSConfig.MinVersion = tls.VersionTLS12
		if value := strings.TrimSpace(serverName); value != "" {
			options.TLSConfig.ServerName = value
		}
		if value := strings.TrimSpace(caFile); value != "" {
			roots, err := loadRootCAs(value)
			if err != nil {
				return nil, err
			}
			options.TLSConfig.RootCAs = roots
		}
	}

	return options, nil
}

func Connect(target, caFile, serverName string, timeout time.Duration) (*redis.Client, error) {
	options, err := Options(target, caFile, serverName)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(options)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redisclient: ping %s failed: %s: %w", SafeTarget(target), sanitizeError(err.Error(), target), ErrUnavailable)
	}

	return client, nil
}

func SafeTarget(target string) string {
	target = strings.TrimSpace(target)
	if target == "" || !strings.Contains(target, "://") {
		return target
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return "redis://invalid"
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func loadRootCAs(path string) (*x509.CertPool, error) {
	pemData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("redisclient: read Redis CA certificate: %w", err)
	}
	roots, err := x509.SystemCertPool()
	if err != nil || roots == nil {
		roots = x509.NewCertPool()
	}
	if !roots.AppendCertsFromPEM(pemData) {
		return nil, fmt.Errorf("redisclient: Redis CA certificate contains no valid certificates")
	}
	return roots, nil
}

func sanitizeError(message, target string) string {
	parsed, err := url.Parse(target)
	if err == nil && parsed.User != nil {
		if password, ok := parsed.User.Password(); ok && password != "" {
			message = strings.ReplaceAll(message, password, "[redacted]")
		}
		if username := parsed.User.Username(); username != "" {
			message = strings.ReplaceAll(message, username, "[redacted]")
		}
	}
	return message
}

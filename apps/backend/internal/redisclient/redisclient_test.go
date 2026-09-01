package redisclient

import (
	"crypto/tls"
	"strings"
	"testing"
	"time"
)

func TestOptionsSupportsLegacyRedisAddress(t *testing.T) {
	options, err := Options("127.0.0.1:6379", "", "")
	if err != nil {
		t.Fatalf("Options: %v", err)
	}
	if options.Addr != "127.0.0.1:6379" || options.TLSConfig != nil {
		t.Fatalf("unexpected legacy options: %#v", options)
	}
}

func TestOptionsSupportsAuthenticatedRediss(t *testing.T) {
	options, err := Options("rediss://default:super-secret@example.cache.amazonaws.com:6379/2", "", "cache.internal")
	if err != nil {
		t.Fatalf("Options: %v", err)
	}
	if options.Addr != "example.cache.amazonaws.com:6379" {
		t.Fatalf("Addr=%q", options.Addr)
	}
	if options.Username != "default" || options.Password != "super-secret" || options.DB != 2 {
		t.Fatalf("authentication/db parsing failed")
	}
	if options.TLSConfig == nil || options.TLSConfig.MinVersion != tls.VersionTLS12 || options.TLSConfig.ServerName != "cache.internal" {
		t.Fatalf("TLS configuration not hardened")
	}
}

func TestTLSOverridesRequireRediss(t *testing.T) {
	if _, err := Options("redis://localhost:6379/0", "", "cache.internal"); err == nil {
		t.Fatal("expected TLS override rejection")
	}
}

func TestSafeTargetRedactsCredentialsAndQuery(t *testing.T) {
	got := SafeTarget("rediss://default:super-secret@example.cache.amazonaws.com:6379/0?read_timeout=1s")
	if got != "rediss://example.cache.amazonaws.com:6379/0" {
		t.Fatalf("SafeTarget=%q", got)
	}
}

func TestConnectErrorDoesNotExposePassword(t *testing.T) {
	_, err := Connect("rediss://default:super-secret@127.0.0.1:1/0", "", "", 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected connection failure")
	}
	if strings.Contains(err.Error(), "super-secret") {
		t.Fatalf("connection error exposed Redis password: %v", err)
	}
}

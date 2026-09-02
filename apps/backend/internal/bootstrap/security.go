package bootstrap

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/mlko01/lyrahcle/apps/backend/internal/agentconfig"
	"github.com/mlko01/lyrahcle/apps/backend/internal/agentidentity"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/keyprovider"
	"github.com/mlko01/lyrahcle/apps/backend/internal/redisclient"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/tokenblocklist"
)

func BuildAgentConfigSigner(cfg config.Config) (*agentconfig.Signer, keyprovider.Provider, error) {
	if cfg.AgentConfigSigningKeyDir == "" && cfg.AgentConfigSigningKeyFile == "" && cfg.AgentConfigSignerEndpoint == "" {
		return nil, nil, nil
	}
	var provider keyprovider.Provider
	var err error
	if cfg.AgentConfigSignerEndpoint != "" {
		provider, err = keyprovider.NewRemoteProvider(cfg.AgentConfigSignerEndpoint, cfg.AgentConfigSignerToken, cfg.AgentConfigSignerKID, cfg.AgentConfigSignerAlgorithm, 5*time.Second)
	} else if cfg.AgentConfigSigningKeyDir != "" {
		provider, err = keyprovider.NewManagedLocalProvider(cfg.AgentConfigSigningKeyDir, cfg.AgentConfigSigningKeyFile)
	} else {
		provider, err = keyprovider.NewLocalProvider(cfg.AgentConfigSigningKeyFile)
	}
	if err != nil {
		return nil, nil, err
	}
	signer, err := agentconfig.NewSigner(provider)
	if err != nil {
		return nil, nil, err
	}
	return signer, provider, nil
}

func BuildAgentCA(cfg config.Config) (*agentidentity.CA, error) {
	if cfg.AgentCACertFile == "" && cfg.AgentCAKeyFile == "" {
		return nil, nil
	}
	return agentidentity.Load(cfg.AgentCACertFile, cfg.AgentCAKeyFile)
}

func ConfigureAgentClientTLS(srv *http.Server, cfg config.Config) error {
	if cfg.AgentCACertFile == "" {
		return nil
	}
	pemBytes, err := os.ReadFile(cfg.AgentCACertFile)
	if err != nil {
		return fmt.Errorf("agent client CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemBytes) {
		return errors.New("agent client CA contains no valid certificates")
	}
	srv.TLSConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		ClientAuth: tls.VerifyClientCertIfGiven,
		ClientCAs:  pool,
	}
	return nil
}

func BuildWebAuthnHandler(cfg config.Config) (*webauthn.WebAuthn, error) {
	return webauthn.New(&webauthn.Config{RPDisplayName: "Lyrahcle", RPID: cfg.WebAuthnRPID, RPOrigins: cfg.WebAuthnOrigins})
}

func ConnectRedis(cfg config.Config) (*redis.Client, error) {
	return redisclient.Connect(cfg.RedisURL, cfg.RedisCACertFile, cfg.RedisTLSServerName, 5*time.Second)
}

func RevocationCheck(users *store.UserStore, blocklist *tokenblocklist.Blocklist) auth.RevocationCheck {
	return func(ctx context.Context, claims *auth.Claims) (bool, error) {
		user, err := users.GetByID(ctx, claims.UserID)
		if err != nil {
			if errors.Is(err, store.ErrUserNotFound) {
				return true, nil
			}
			return false, err
		}
		if !user.Enabled || user.Role != string(claims.Role) || user.AuthVersion != claims.AuthVersion {
			return true, nil
		}
		if claims.OrganizationID != uuid.Nil && user.OrganizationID != claims.OrganizationID {
			return true, nil
		}
		if user.MustChangePassword != claims.PasswordChangeRequired {
			return true, nil
		}
		issuedAt := time.Unix(0, claims.SessionIssuedAt).UTC()
		return blocklist.Check(ctx, claims.ID, claims.UserID.String(), issuedAt)
	}
}

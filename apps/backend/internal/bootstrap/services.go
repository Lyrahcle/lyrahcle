package bootstrap

import (
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/redis/go-redis/v9"

	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/mailer"
	"github.com/mlko01/lyrahcle/apps/backend/internal/respcache"
	"github.com/mlko01/lyrahcle/apps/backend/internal/sessionstore"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/tokenblocklist"
	"github.com/mlko01/lyrahcle/apps/backend/internal/webauthnflow"
)

type RequestServices struct {
	Blocklist       *tokenblocklist.Blocklist
	Sessions        *sessionstore.Store
	Cache           *respcache.Cache
	WebAuthnFlows   *webauthnflow.Store
	WebAuthnHandler *webauthn.WebAuthn
	MailClient      *mailer.Client
	Revocation      auth.RevocationCheck
	RateLimiters    RateLimiters
}

func BuildRequestServices(cfg config.Config, users *store.UserStore, redisClient *redis.Client) (RequestServices, error) {
	blocklist := tokenblocklist.New(redisClient)
	limiters, err := BuildRateLimiters(cfg, redisClient)
	if err != nil {
		return RequestServices{}, err
	}
	webAuthnHandler, err := BuildWebAuthnHandler(cfg)
	if err != nil {
		return RequestServices{}, err
	}
	return RequestServices{
		Blocklist:       blocklist,
		Sessions:        sessionstore.New(redisClient),
		Cache:           respcache.New(redisClient),
		WebAuthnFlows:   webauthnflow.New(redisClient),
		WebAuthnHandler: webAuthnHandler,
		MailClient:      mailer.New(mailer.Config{Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUser, Password: cfg.SMTPPassword, From: cfg.SMTPFrom, ServerName: cfg.SMTPServerName}),
		Revocation:      RevocationCheck(users, blocklist),
		RateLimiters:    limiters,
	}, nil
}

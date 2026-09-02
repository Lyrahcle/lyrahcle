package bootstrap

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
	"github.com/redis/go-redis/v9"

	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/ratelimit"
)

type RateLimiters struct {
	Global func(http.Handler) http.Handler
	Login  func(http.Handler) http.Handler
	Tenant *ratelimit.Limiter
}

func BuildRateLimiters(cfg config.Config, redisClient *redis.Client) (RateLimiters, error) {
	global := httprate.Limit(cfg.RateLimitPlatformCeilingPerMinute, time.Minute, httprate.WithKeyByIP())
	login := httprate.Limit(10, time.Minute, httprate.WithKeyByIP())
	if redisClient != nil {
		global = httprate.Limit(cfg.RateLimitPlatformCeilingPerMinute, time.Minute, httprate.WithKeyByIP(), httprate.WithLimitCounter(ratelimit.NewRedisCounter(redisClient, "platform-ceiling")))
		login = httprate.Limit(10, time.Minute, httprate.WithKeyByIP(), httprate.WithLimitCounter(ratelimit.NewRedisCounter(redisClient, "login")))
	}
	tenant, err := ratelimit.NewTenantLimiter(redisClient, ratelimit.Policy{
		Window:           time.Minute,
		TenantReadLimit:  cfg.RateLimitTenantReadPerMinute,
		ActorReadLimit:   cfg.RateLimitActorReadPerMinute,
		TenantWriteLimit: cfg.RateLimitTenantWritePerMinute,
		ActorWriteLimit:  cfg.RateLimitActorWritePerMinute,
		TenantAgentLimit: cfg.RateLimitTenantAgentPerMinute,
		AgentLimit:       cfg.RateLimitAgentPerMinute,
	})
	if err != nil {
		return RateLimiters{}, err
	}
	return RateLimiters{Global: global, Login: login, Tenant: tenant}, nil
}

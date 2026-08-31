package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

type rateLimitStatusResponse struct {
	WindowSeconds                  int  `json:"window_seconds"`
	PlatformCeilingPerMinute       int  `json:"platform_ceiling_per_minute"`
	TenantReadPerMinute            int  `json:"tenant_read_per_minute"`
	ActorReadPerMinute             int  `json:"actor_read_per_minute"`
	TenantWritePerMinute           int  `json:"tenant_write_per_minute"`
	ActorWritePerMinute            int  `json:"actor_write_per_minute"`
	TenantAgentPerMinute           int  `json:"tenant_agent_per_minute"`
	AgentPerMinute                 int  `json:"agent_per_minute"`
	RedisDistributed               bool `json:"redis_distributed"`
	HumanAndMachineBudgetsSeparate bool `json:"human_and_machine_budgets_separate"`
}

func registerRateLimitStatusRoutes(
	r chi.Router,
	cfg config.Config,
	users *store.UserStore,
	isTokenRevoked func(context.Context, *auth.Claims) (bool, error),
	redisDistributed bool,
) {
	requirePlatformAdmin := func(next http.Handler) http.Handler {
		return auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			allowed, err := users.IsPlatformAdministrator(r.Context(), claims.UserID)
			if err != nil {
				http.Error(w, "platform authorization unavailable", http.StatusInternalServerError)
				return
			}
			if !allowed {
				http.Error(w, "platform administrator required", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		}))
	}

	r.With(requirePlatformAdmin).Get("/platform/rate-limits", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rateLimitStatusResponse{
			WindowSeconds:                  60,
			PlatformCeilingPerMinute:       cfg.RateLimitPlatformCeilingPerMinute,
			TenantReadPerMinute:            cfg.RateLimitTenantReadPerMinute,
			ActorReadPerMinute:             cfg.RateLimitActorReadPerMinute,
			TenantWritePerMinute:           cfg.RateLimitTenantWritePerMinute,
			ActorWritePerMinute:            cfg.RateLimitActorWritePerMinute,
			TenantAgentPerMinute:           cfg.RateLimitTenantAgentPerMinute,
			AgentPerMinute:                 cfg.RateLimitAgentPerMinute,
			RedisDistributed:               redisDistributed,
			HumanAndMachineBudgetsSeparate: true,
		})
	})
}

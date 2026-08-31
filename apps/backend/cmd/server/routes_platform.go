package main

import (
	"github.com/go-chi/chi/v5"
)

func (a *apiRoutes) registerPlatformRoutes(r chi.Router) {
	agentConfigKeyProvider := a.agentConfigKeyProvider
	auditLog := a.auditLog
	cfg := a.cfg
	isTokenRevoked := a.isTokenRevoked
	organizations := a.organizations
	rbac := a.rbac
	redisClient := a.redisClient
	tenantScope := a.tenantScope
	users := a.users

	registerOrganizationRoutes(r, cfg, organizations, users, auditLog, isTokenRevoked)
	registerSigningKeyRoutes(r, cfg, agentConfigKeyProvider, users, auditLog, isTokenRevoked)
	registerRateLimitStatusRoutes(r, cfg, users, isTokenRevoked, redisClient != nil)
	registerSecretStatusRoutes(r, cfg, users, isTokenRevoked)
	registerTenantScopeRoutes(r, cfg, tenantScope, rbac, isTokenRevoked)
	registerRBACRoutes(r, cfg, rbac, auditLog, isTokenRevoked)
}

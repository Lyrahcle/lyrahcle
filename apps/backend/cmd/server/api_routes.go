package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/redis/go-redis/v9"

	"github.com/mlko01/lyrahcle/apps/backend/internal/agentconfig"
	"github.com/mlko01/lyrahcle/apps/backend/internal/agentidentity"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	incidentdomain "github.com/mlko01/lyrahcle/apps/backend/internal/incident"
	"github.com/mlko01/lyrahcle/apps/backend/internal/keyprovider"
	"github.com/mlko01/lyrahcle/apps/backend/internal/mailer"
	"github.com/mlko01/lyrahcle/apps/backend/internal/ratelimit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/respcache"
	"github.com/mlko01/lyrahcle/apps/backend/internal/sessionstore"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/tokenblocklist"
	"github.com/mlko01/lyrahcle/apps/backend/internal/webauthnflow"
)

type apiRoutes struct {
	cfg config.Config

	assets          *store.AssetStore
	programs        *store.ProgramStore
	rf              *store.RFStore
	rfAnomalies     *store.AnomalyStore
	satellites      *store.SatelliteStore
	users           *store.UserStore
	agentKeys       *store.AgentKeyStore
	auditLog        *store.AuditStore
	spaceCVEs       *store.SpaceCVEStore
	assetAlerts     *store.AssetAlertStore
	incidents       *store.IncidentStore
	incidentService *incidentdomain.Service
	rbac            *store.RBACStore

	redisClient            *redis.Client
	agentConfigSigner      *agentconfig.Signer
	agentConfigKeyProvider keyprovider.Provider
	detectionRules         *store.DetectionRuleStore
	blocklist              *tokenblocklist.Blocklist
	sessions               *sessionstore.Store
	cache                  *respcache.Cache
	webAuthnCredentials    *store.WebAuthnStore
	passwordResets         *store.PasswordResetStore
	searchStore            *store.SearchStore
	agentIdentities        *store.AgentIdentityStore
	agentConfigs           *store.AgentConfigStore
	fleetRollouts          *store.FleetRolloutStore
	orbitalEvents          *store.OrbitalEventStore
	organizations          *store.OrganizationStore
	tenantScope            *store.TenantScopeStore
	agentCA                *agentidentity.CA
	webAuthnFlows          *webauthnflow.Store
	webAuthnHandler        *webauthn.WebAuthn
	mailClient             *mailer.Client
	isTokenRevoked         auth.RevocationCheck
	loginLimiter           func(http.Handler) http.Handler
	tenantLimiter          *ratelimit.Limiter
	perm                   permissionMiddleware
}

func (a *apiRoutes) register(r chi.Router) {
	a.registerCoreRoutes(r)
	a.registerAssetProgramRoutes(r)
	a.registerAuthenticationRoutes(r)
	a.registerPlatformRoutes(r)
	a.registerUserRoutes(r)
	a.registerAgentKeyDeviceRoutes(r)
	a.registerDetectionRFRoutes(r)
	a.registerSatelliteTMTCRoutes(r)
	a.registerAuditThreatRoutes(r)
	a.registerIncidentAlertRoutes(r)
}

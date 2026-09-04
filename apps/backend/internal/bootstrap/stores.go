package bootstrap

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

type Stores struct {
	Assets               *store.AssetStore
	Programs             *store.ProgramStore
	RF                   *store.RFStore
	RFAnomalies          *store.AnomalyStore
	Satellites           *store.SatelliteStore
	Users                *store.UserStore
	AgentKeys            *store.AgentKeyStore
	AuditLog             *store.AuditStore
	SpaceCVEs            *store.SpaceCVEStore
	AssetAlerts          *store.AssetAlertStore
	Incidents            *store.IncidentStore
	RBAC                 *store.RBACStore
	DetectionRules       *store.DetectionRuleStore
	WebAuthnCredentials  *store.WebAuthnStore
	PasswordResets       *store.PasswordResetStore
	Search               *store.SearchStore
	AgentIdentities      *store.AgentIdentityStore
	AgentConfigs         *store.AgentConfigStore
	FleetRollouts        *store.FleetRolloutStore
	OrbitalEvents        *store.OrbitalEventStore
	Organizations        *store.OrganizationStore
	TenantScope          *store.TenantScopeStore
	ExternalObservations *store.ExternalObservationStore
}

func NewStores(pool *pgxpool.Pool) Stores {
	return Stores{
		Assets:               store.NewAssetStore(pool),
		Programs:             store.NewProgramStore(pool),
		RF:                   store.NewRFStore(pool),
		RFAnomalies:          store.NewAnomalyStore(pool),
		Satellites:           store.NewSatelliteStore(pool),
		Users:                store.NewUserStore(pool),
		AgentKeys:            store.NewAgentKeyStore(pool),
		AuditLog:             store.NewAuditStore(pool),
		SpaceCVEs:            store.NewSpaceCVEStore(pool),
		AssetAlerts:          store.NewAssetAlertStore(pool),
		Incidents:            store.NewIncidentStore(pool),
		RBAC:                 store.NewRBACStore(pool),
		DetectionRules:       store.NewDetectionRuleStore(pool),
		WebAuthnCredentials:  store.NewWebAuthnStore(pool),
		PasswordResets:       store.NewPasswordResetStore(pool),
		Search:               store.NewSearchStore(pool),
		AgentIdentities:      store.NewAgentIdentityStore(pool),
		AgentConfigs:         store.NewAgentConfigStore(pool),
		FleetRollouts:        store.NewFleetRolloutStore(pool),
		OrbitalEvents:        store.NewOrbitalEventStore(pool),
		Organizations:        store.NewOrganizationStore(pool),
		TenantScope:          store.NewTenantScopeStore(pool),
		ExternalObservations: store.NewExternalObservationStore(pool),
	}
}

package audit

const (
	ActionLogin                 = "auth.login"
	ActionLogout                = "auth.logout"
	ActionLoginFailed           = "auth.login_failed"
	ActionRegister              = "auth.register"
	ActionUserCreated           = "user.created"
	ActionTOTPEnabled           = "auth.totp_enabled"
	ActionTOTPDisabled          = "auth.totp_disabled"
	ActionPasswordChanged       = "auth.password_changed"
	ActionAgentKeyCreated       = "agent_key.created"
	ActionAgentKeyRevoked       = "agent_key.revoked"
	ActionAgentKeyAuthFailed    = "agent_key.auth_failed"
	ActionSatelliteUpserted     = "satellite.upserted"
	ActionRoleChanged           = "user.role_changed"
	ActionAssetAlertCreated     = "asset_alert.created"
	ActionAssetAlertAcked       = "asset_alert.acknowledged"
	ActionIncidentCreated       = "incident.created"
	ActionIncidentUpdated       = "incident.updated"
	ActionIncidentNoteAdded     = "incident.note_added"
	ActionIncidentEvidenceAdded = "incident.evidence_added"

	ActorUser      = "user"
	ActorAgent     = "agent"
	ActorAnonymous = "anonymous"
	ActorSystem    = "system"
)

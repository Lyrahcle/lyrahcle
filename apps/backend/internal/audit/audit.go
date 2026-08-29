package audit

const (
	ActionLogin              = "auth.login"
	ActionLogout             = "auth.logout"
	ActionLoginFailed        = "auth.login_failed"
	ActionRegister           = "auth.register"
	ActionPasswordChanged    = "auth.password_changed"
	ActionAgentKeyCreated    = "agent_key.created"
	ActionAgentKeyRevoked    = "agent_key.revoked"
	ActionAgentKeyAuthFailed = "agent_key.auth_failed"
	ActionSatelliteUpserted  = "satellite.upserted"
	ActionRoleChanged        = "user.role_changed"

	ActorUser      = "user"
	ActorAgent     = "agent"
	ActorAnonymous = "anonymous"
)

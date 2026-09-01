package config

import (
	"context"
	"errors"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mlko01/lyrahcle/apps/backend/internal/secretprovider"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
)

type Config struct {
	DatabaseURL                       string
	RedisURL                          string
	RedisCACertFile                   string
	RedisTLSServerName                string
	Port                              string
	CelestrakGroups                   []string
	TLECorrelationMaxAge              time.Duration
	TLEStaleMaxAge                    time.Duration
	OrbitalAutoEscalation             bool
	JWTSecret                         []byte
	JWTPreviousSecret                 []byte
	JWTAccessTTL                      time.Duration
	TLSCertFile                       string
	TLSKeyFile                        string
	SeedAdminEmail                    string
	SeedAdminPass                     string
	Environment                       string
	WebAuthnRPID                      string
	WebAuthnOrigins                   []string
	CORSAllowedOrigins                []string
	PasswordRecoveryEnabled           bool
	PasswordResetBaseURL              string
	SMTPHost                          string
	SMTPPort                          string
	SMTPUser                          string
	SMTPPassword                      string
	SMTPFrom                          string
	SMTPServerName                    string
	OTLPTraceEndpoint                 string
	AgentCACertFile                   string
	AgentCAKeyFile                    string
	AgentCertificateTTL               time.Duration
	AgentEnrollmentTTL                time.Duration
	AgentConfigSigningKeyFile         string
	AgentConfigSigningKeyDir          string
	AgentConfigSignerEndpoint         string
	AgentConfigSignerToken            string
	AgentConfigSignerKID              string
	AgentConfigSignerAlgorithm        string
	AgentConfigSigningGracePeriod     time.Duration
	RateLimitPlatformCeilingPerMinute int
	RateLimitTenantReadPerMinute      int
	RateLimitActorReadPerMinute       int
	RateLimitTenantWritePerMinute     int
	RateLimitActorWritePerMinute      int
	RateLimitTenantAgentPerMinute     int
	RateLimitAgentPerMinute           int
	SecretProviderName                string
	SecretProviderExternal            bool
	SecretProviderLocation            string
	SecretProviderHealthy             bool
}

func Load() Config {
	environment := getEnv("ENV", "development")
	production := strings.EqualFold(environment, "production") || strings.EqualFold(environment, "prod")
	secretSource := loadSecretProvider(production)
	providerInfo := secretSource.Info()
	providerHealth := secretSource.Health(context.Background())
	if production && !providerHealth.Healthy {
		log.Fatalf("FATAL: secret provider %s is unhealthy: %s", providerInfo.Name, providerHealth.Message)
	}

	secret := getSecret(secretSource, "JWT_SECRET", "")
	insecureDefault := false
	if secret == "" {
		secret = "insecure-dev-secret-change-me"
		insecureDefault = true
	}

	ttlRaw := getEnv("JWT_ACCESS_TTL", "30m")
	ttl, err := time.ParseDuration(ttlRaw)
	if err != nil || ttl < 5*time.Minute || ttl > 24*time.Hour {
		log.Fatalf("FATAL: JWT_ACCESS_TTL must be a duration between 5m and 24h")
	}
	agentCertTTL, err := time.ParseDuration(getEnv("AGENT_CERTIFICATE_TTL", "168h"))
	if err != nil || agentCertTTL < time.Hour || agentCertTTL > 2160*time.Hour {
		log.Fatal("FATAL: AGENT_CERTIFICATE_TTL must be between 1h and 2160h")
	}
	agentEnrollmentTTL, err := time.ParseDuration(getEnv("AGENT_ENROLLMENT_TTL", "15m"))
	if err != nil || agentEnrollmentTTL < 5*time.Minute || agentEnrollmentTTL > time.Hour {
		log.Fatal("FATAL: AGENT_ENROLLMENT_TTL must be between 5m and 1h")
	}
	agentConfigSigningGracePeriod, err := time.ParseDuration(getEnv("AGENT_CONFIG_SIGNING_GRACE_PERIOD", "168h"))
	if err != nil || agentConfigSigningGracePeriod < time.Hour || agentConfigSigningGracePeriod > 30*24*time.Hour {
		log.Fatal("FATAL: AGENT_CONFIG_SIGNING_GRACE_PERIOD must be between 1h and 720h")
	}
	tLECorrelationMaxAge, err := time.ParseDuration(getEnv("TLE_CORRELATION_MAX_AGE", "24h"))
	if err != nil || tLECorrelationMaxAge < time.Hour || tLECorrelationMaxAge > 7*24*time.Hour {
		log.Fatal("FATAL: TLE_CORRELATION_MAX_AGE must be between 1h and 168h")
	}
	tLEStaleMaxAge, err := time.ParseDuration(getEnv("TLE_STALE_MAX_AGE", "72h"))
	if err != nil || tLEStaleMaxAge <= tLECorrelationMaxAge || tLEStaleMaxAge > 30*24*time.Hour {
		log.Fatal("FATAL: TLE_STALE_MAX_AGE must be greater than TLE_CORRELATION_MAX_AGE and no more than 720h")
	}

	cfg := Config{
		DatabaseURL:                       getSecret(secretSource, "DATABASE_URL", "postgres://lyrahcle:lyrahcle@localhost:5432/lyrahcle?sslmode=disable"),
		RedisURL:                          getSecret(secretSource, "REDIS_URL", "redis://localhost:6379/0"),
		RedisCACertFile:                   getEnv("REDIS_CA_CERT_FILE", ""),
		RedisTLSServerName:                getEnv("REDIS_TLS_SERVER_NAME", ""),
		Port:                              getEnv("PORT", "8080"),
		CelestrakGroups:                   strings.Split(getEnv("CELESTRAK_GROUPS", "stations"), ","),
		TLECorrelationMaxAge:              tLECorrelationMaxAge,
		TLEStaleMaxAge:                    tLEStaleMaxAge,
		OrbitalAutoEscalation:             strings.EqualFold(getEnv("ORBITAL_AUTO_ESCALATION_ENABLED", "false"), "true"),
		JWTSecret:                         []byte(secret),
		JWTPreviousSecret:                 []byte(getSecret(secretSource, "JWT_SECRET_PREVIOUS", "")),
		JWTAccessTTL:                      ttl,
		TLSCertFile:                       getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:                        getEnv("TLS_KEY_FILE", ""),
		SeedAdminEmail:                    getEnv("SEED_ADMIN_EMAIL", "admin"),
		SeedAdminPass:                     getSecret(secretSource, "SEED_ADMIN_PASSWORD", "admin"),
		Environment:                       environment,
		WebAuthnRPID:                      getEnv("WEBAUTHN_RP_ID", "localhost"),
		WebAuthnOrigins:                   splitCSV(getEnv("WEBAUTHN_ORIGINS", "http://localhost:5173")),
		CORSAllowedOrigins:                splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "")),
		PasswordRecoveryEnabled:           strings.EqualFold(getEnv("PASSWORD_RECOVERY_ENABLED", "false"), "true"),
		PasswordResetBaseURL:              getEnv("PASSWORD_RESET_BASE_URL", "http://localhost:5173/"),
		SMTPHost:                          getEnv("SMTP_HOST", ""),
		SMTPPort:                          getEnv("SMTP_PORT", "465"),
		SMTPUser:                          getEnv("SMTP_USER", ""),
		SMTPPassword:                      getSecret(secretSource, "SMTP_PASSWORD", ""),
		SMTPFrom:                          getEnv("SMTP_FROM", ""),
		SMTPServerName:                    getEnv("SMTP_SERVER_NAME", ""),
		OTLPTraceEndpoint:                 getEnv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", ""),
		AgentCACertFile:                   getEnv("AGENT_CA_CERT_FILE", ""),
		AgentCAKeyFile:                    getEnv("AGENT_CA_KEY_FILE", ""),
		AgentCertificateTTL:               agentCertTTL,
		AgentEnrollmentTTL:                agentEnrollmentTTL,
		AgentConfigSigningKeyFile:         getEnv("AGENT_CONFIG_SIGNING_KEY_FILE", ""),
		AgentConfigSigningKeyDir:          getEnv("AGENT_CONFIG_SIGNING_KEY_DIR", ""),
		AgentConfigSignerEndpoint:         getEnv("AGENT_CONFIG_SIGNER_ENDPOINT", ""),
		AgentConfigSignerToken:            getSecret(secretSource, "AGENT_CONFIG_SIGNER_TOKEN", ""),
		AgentConfigSignerKID:              getEnv("AGENT_CONFIG_SIGNER_KID", ""),
		AgentConfigSignerAlgorithm:        getEnv("AGENT_CONFIG_SIGNER_ALGORITHM", "Ed25519"),
		AgentConfigSigningGracePeriod:     agentConfigSigningGracePeriod,
		RateLimitPlatformCeilingPerMinute: envInt("RATE_LIMIT_PLATFORM_CEILING_PER_MINUTE", 50000, 1000, 500000),
		RateLimitTenantReadPerMinute:      envInt("RATE_LIMIT_TENANT_READ_PER_MINUTE", 1200, 10, 100000),
		RateLimitActorReadPerMinute:       envInt("RATE_LIMIT_ACTOR_READ_PER_MINUTE", 300, 5, 50000),
		RateLimitTenantWritePerMinute:     envInt("RATE_LIMIT_TENANT_WRITE_PER_MINUTE", 300, 5, 50000),
		RateLimitActorWritePerMinute:      envInt("RATE_LIMIT_ACTOR_WRITE_PER_MINUTE", 120, 2, 20000),
		RateLimitTenantAgentPerMinute:     envInt("RATE_LIMIT_TENANT_AGENT_PER_MINUTE", 12000, 60, 500000),
		RateLimitAgentPerMinute:           envInt("RATE_LIMIT_AGENT_PER_MINUTE", 3000, 30, 100000),
		SecretProviderName:                providerInfo.Name,
		SecretProviderExternal:            providerInfo.External,
		SecretProviderLocation:            providerInfo.Location,
		SecretProviderHealthy:             providerHealth.Healthy,
	}

	if cfg.RateLimitActorReadPerMinute > cfg.RateLimitTenantReadPerMinute || cfg.RateLimitActorWritePerMinute > cfg.RateLimitTenantWritePerMinute || cfg.RateLimitAgentPerMinute > cfg.RateLimitTenantAgentPerMinute {
		log.Fatal("FATAL: actor/agent rate limits must not exceed their tenant budgets")
	}
	maxTenantBudget := cfg.RateLimitTenantReadPerMinute
	if cfg.RateLimitTenantWritePerMinute > maxTenantBudget {
		maxTenantBudget = cfg.RateLimitTenantWritePerMinute
	}
	if cfg.RateLimitTenantAgentPerMinute > maxTenantBudget {
		maxTenantBudget = cfg.RateLimitTenantAgentPerMinute
	}
	if cfg.RateLimitPlatformCeilingPerMinute <= maxTenantBudget {
		log.Fatal("FATAL: RATE_LIMIT_PLATFORM_CEILING_PER_MINUTE must exceed every tenant-scoped budget")
	}

	production = cfg.IsProduction()
	if production {
		if err := validateProductionRedisURL(cfg.RedisURL); err != nil {
			log.Fatalf("FATAL: %v", err)
		}
	}
	for _, origin := range cfg.CORSAllowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			log.Fatal("FATAL: CORS_ALLOWED_ORIGINS must not contain wildcard '*'")
		}
		if !strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://") {
			log.Fatalf("FATAL: invalid CORS origin %q; expected absolute http(s) origin", origin)
		}
		if production && !strings.HasPrefix(strings.ToLower(origin), "https://") {
			log.Fatalf("FATAL: CORS origin %q must use HTTPS in production", origin)
		}
	}
	if production && cfg.SecretProviderName == "env" {
		log.Fatal("FATAL: ENV=production requires SECRET_PROVIDER=file or SECRET_PROVIDER=remote; process-environment secret custody is not allowed")
	}
	if cfg.OrbitalAutoEscalation {
		log.Println("WARNING: orbital automatic severity escalation is enabled; validate RF/orbital performance against representative station data before production use")
	}

	if insecureDefault {
		if production {
			log.Fatal("FATAL: ENV=production but JWT_SECRET is not set. Refusing to start with an insecure default secret.")
		}
		log.Println("WARNING: JWT_SECRET is not set; using an insecure default. Set JWT_SECRET in production.")
	}
	if len(cfg.JWTSecret) < 32 {
		if production {
			log.Fatal("FATAL: JWT_SECRET must contain at least 32 bytes in production.")
		}
		log.Println("WARNING: JWT_SECRET is shorter than 32 bytes.")
	}
	if production && cfg.JWTAccessTTL > 2*time.Hour {
		log.Fatal("FATAL: JWT_ACCESS_TTL must not exceed 2h in production.")
	}
	if len(cfg.JWTPreviousSecret) > 0 && len(cfg.JWTPreviousSecret) < 32 {
		if production {
			log.Fatal("FATAL: JWT_SECRET_PREVIOUS must contain at least 32 bytes when configured in production.")
		}
		log.Println("WARNING: JWT_SECRET_PREVIOUS is shorter than 32 bytes.")
	}

	if cfg.SeedAdminPass == "admin" {
		if production {
			log.Fatal("FATAL: ENV=production but SEED_ADMIN_PASSWORD is unset or still the default 'admin'. Refusing to start.")
		}
		log.Println("WARNING: SEED_ADMIN_PASSWORD is unset or still the default 'admin'. Set a strong value in production.")
	}
	if production {
		if err := validate.Password(cfg.SeedAdminPass); err != nil {
			log.Fatalf("FATAL: SEED_ADMIN_PASSWORD does not meet the password policy: %v", err)
		}
	}

	if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
		if production {
			log.Fatal("FATAL: ENV=production but TLS_CERT_FILE/TLS_KEY_FILE are not set. Refusing to serve plain HTTP in production.")
		}
		log.Println("WARNING: TLS_CERT_FILE/TLS_KEY_FILE not set; serving plain HTTP. Set both to enable HTTPS (see scripts/generate-dev-certs.sh).")
	}

	if cfg.WebAuthnRPID == "" || len(cfg.WebAuthnOrigins) == 0 {
		log.Fatal("FATAL: WEBAUTHN_RP_ID and WEBAUTHN_ORIGINS must be configured")
	}
	if production {
		for _, origin := range cfg.WebAuthnOrigins {
			if !strings.HasPrefix(strings.ToLower(origin), "https://") {
				log.Fatal("FATAL: WEBAUTHN_ORIGINS must use HTTPS in production")
			}
		}
	}
	if cfg.PasswordRecoveryEnabled {
		if cfg.SMTPHost == "" || cfg.SMTPFrom == "" || cfg.SMTPServerName == "" || cfg.PasswordResetBaseURL == "" {
			log.Fatal("FATAL: password recovery requires SMTP_HOST, SMTP_FROM, SMTP_SERVER_NAME, and PASSWORD_RESET_BASE_URL")
		}
		if production && !strings.HasPrefix(strings.ToLower(cfg.PasswordResetBaseURL), "https://") {
			log.Fatal("FATAL: PASSWORD_RESET_BASE_URL must use HTTPS in production")
		}
	}
	if production && cfg.OTLPTraceEndpoint != "" && !strings.HasPrefix(strings.ToLower(cfg.OTLPTraceEndpoint), "https://") {
		log.Fatal("FATAL: OTEL_EXPORTER_OTLP_TRACES_ENDPOINT must use HTTPS in production")
	}

	if (cfg.AgentCACertFile == "") != (cfg.AgentCAKeyFile == "") {
		log.Fatal("FATAL: AGENT_CA_CERT_FILE and AGENT_CA_KEY_FILE must be configured together")
	}
	if cfg.AgentCACertFile != "" && (cfg.TLSCertFile == "" || cfg.TLSKeyFile == "") {
		log.Fatal("FATAL: SDR agent mTLS requires TLS_CERT_FILE and TLS_KEY_FILE")
	}
	if production && (cfg.AgentCACertFile == "" || cfg.AgentCAKeyFile == "") {
		log.Println("WARNING: agent mTLS enrollment is disabled because AGENT_CA_CERT_FILE/AGENT_CA_KEY_FILE are not configured")
	}
	signerConfigured := cfg.AgentConfigSigningKeyFile != "" || cfg.AgentConfigSigningKeyDir != "" || cfg.AgentConfigSignerEndpoint != ""
	if signerConfigured && cfg.AgentCACertFile == "" {
		log.Fatal("FATAL: agent config signing requires agent mTLS enrollment")
	}
	if cfg.AgentConfigSignerEndpoint != "" && (cfg.AgentConfigSigningKeyFile != "" || cfg.AgentConfigSigningKeyDir != "") {
		log.Fatal("FATAL: external remote signer cannot be combined with local agent-config signing key settings")
	}
	if cfg.AgentConfigSignerEndpoint != "" && (cfg.AgentConfigSignerKID == "" || !strings.HasPrefix(strings.ToLower(cfg.AgentConfigSignerEndpoint), "https://")) {
		log.Fatal("FATAL: AGENT_CONFIG_SIGNER_ENDPOINT requires HTTPS and AGENT_CONFIG_SIGNER_KID")
	}
	if production && !signerConfigured {
		log.Println("WARNING: secure remote agent configuration is disabled because AGENT_CONFIG_SIGNING_KEY_FILE/AGENT_CONFIG_SIGNING_KEY_DIR are not configured")
	}

	return cfg
}

func (c Config) IsProduction() bool {
	return strings.EqualFold(c.Environment, "production") || strings.EqualFold(c.Environment, "prod")
}

func validateProductionRedisURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !strings.EqualFold(parsed.Scheme, "rediss") || parsed.Host == "" {
		return errors.New("production REDIS_URL must be a valid rediss:// URL")
	}
	if parsed.User == nil {
		return errors.New("production REDIS_URL must include Redis authentication")
	}
	password, ok := parsed.User.Password()
	if !ok || strings.TrimSpace(password) == "" {
		return errors.New("production REDIS_URL must include a non-empty Redis password or AUTH token")
	}
	return nil
}

func loadSecretProvider(production bool) secretprovider.Provider {
	mode := strings.ToLower(strings.TrimSpace(getEnv("SECRET_PROVIDER", "env")))
	switch mode {
	case "env":
		return secretprovider.EnvProvider{}
	case "file":
		dir := strings.TrimSpace(getEnv("SECRET_FILE_DIR", ""))
		if dir == "" {
			log.Fatal("FATAL: SECRET_PROVIDER=file requires SECRET_FILE_DIR")
		}
		return secretprovider.FileProvider{Dir: dir}
	case "remote":
		endpoint := strings.TrimSpace(getEnv("SECRET_PROVIDER_ENDPOINT", ""))
		if endpoint == "" {
			log.Fatal("FATAL: SECRET_PROVIDER=remote requires SECRET_PROVIDER_ENDPOINT")
		}
		token := strings.TrimSpace(getEnv("SECRET_PROVIDER_TOKEN", ""))
		if tokenFile := strings.TrimSpace(getEnv("SECRET_PROVIDER_TOKEN_FILE", "")); token == "" && tokenFile != "" {
			b, err := os.ReadFile(tokenFile)
			if err != nil {
				log.Fatalf("FATAL: read SECRET_PROVIDER_TOKEN_FILE: %v", err)
			}
			token = strings.TrimSpace(string(b))
		}
		provider := secretprovider.RemoteProvider{Endpoint: endpoint, Token: token}
		if production && !provider.Health(context.Background()).Healthy {
			log.Fatal("FATAL: remote secret provider health check failed")
		}
		return provider
	default:
		log.Fatalf("FATAL: unsupported SECRET_PROVIDER %q (expected env, file, or remote)", mode)
	}
	return secretprovider.EnvProvider{}
}

func getSecret(provider secretprovider.Provider, key, fallback string) string {
	value, err := provider.Get(context.Background(), key)
	if err == nil {
		return value
	}
	if errors.Is(err, secretprovider.ErrNotFound) {
		return fallback
	}
	log.Fatalf("FATAL: resolve secret %s from %s provider: %v", key, provider.Info().Name, err)
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func envInt(key string, fallback, min, max int) int {
	raw := getEnv(key, strconv.Itoa(fallback))
	value, err := strconv.Atoi(raw)
	if err != nil || value < min || value > max {
		log.Fatalf("FATAL: %s must be an integer between %d and %d", key, min, max)
	}
	return value
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

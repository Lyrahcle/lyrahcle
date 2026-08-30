package config

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
)

type Config struct {
	DatabaseURL               string
	RedisURL                  string
	Port                      string
	CelestrakGroups           []string
	TLECorrelationMaxAge      time.Duration
	TLEStaleMaxAge            time.Duration
	OrbitalAutoEscalation     bool
	JWTSecret                 []byte
	JWTPreviousSecret         []byte
	JWTAccessTTL              time.Duration
	TLSCertFile               string
	TLSKeyFile                string
	SeedAdminEmail            string
	SeedAdminPass             string
	Environment               string
	WebAuthnRPID              string
	WebAuthnOrigins           []string
	CORSAllowedOrigins        []string
	PasswordRecoveryEnabled   bool
	PasswordResetBaseURL      string
	SMTPHost                  string
	SMTPPort                  string
	SMTPUser                  string
	SMTPPassword              string
	SMTPFrom                  string
	SMTPServerName            string
	OTLPTraceEndpoint         string
	AgentCACertFile           string
	AgentCAKeyFile            string
	AgentCertificateTTL       time.Duration
	AgentEnrollmentTTL        time.Duration
	AgentConfigSigningKeyFile string
}

func Load() Config {
	secret := getEnv("JWT_SECRET", "")
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
	tLECorrelationMaxAge, err := time.ParseDuration(getEnv("TLE_CORRELATION_MAX_AGE", "24h"))
	if err != nil || tLECorrelationMaxAge < time.Hour || tLECorrelationMaxAge > 7*24*time.Hour {
		log.Fatal("FATAL: TLE_CORRELATION_MAX_AGE must be between 1h and 168h")
	}
	tLEStaleMaxAge, err := time.ParseDuration(getEnv("TLE_STALE_MAX_AGE", "72h"))
	if err != nil || tLEStaleMaxAge <= tLECorrelationMaxAge || tLEStaleMaxAge > 30*24*time.Hour {
		log.Fatal("FATAL: TLE_STALE_MAX_AGE must be greater than TLE_CORRELATION_MAX_AGE and no more than 720h")
	}

	cfg := Config{
		DatabaseURL:               getEnv("DATABASE_URL", "postgres://lyrahcle:lyrahcle@localhost:5432/lyrahcle?sslmode=disable"),
		RedisURL:                  getEnv("REDIS_URL", "localhost:6379"),
		Port:                      getEnv("PORT", "8080"),
		CelestrakGroups:           strings.Split(getEnv("CELESTRAK_GROUPS", "stations"), ","),
		TLECorrelationMaxAge:      tLECorrelationMaxAge,
		TLEStaleMaxAge:            tLEStaleMaxAge,
		OrbitalAutoEscalation:     strings.EqualFold(getEnv("ORBITAL_AUTO_ESCALATION_ENABLED", "false"), "true"),
		JWTSecret:                 []byte(secret),
		JWTPreviousSecret:         []byte(getEnv("JWT_SECRET_PREVIOUS", "")),
		JWTAccessTTL:              ttl,
		TLSCertFile:               getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:                getEnv("TLS_KEY_FILE", ""),
		SeedAdminEmail:            getEnv("SEED_ADMIN_EMAIL", "admin"),
		SeedAdminPass:             getEnv("SEED_ADMIN_PASSWORD", "admin"),
		Environment:               getEnv("ENV", "development"),
		WebAuthnRPID:              getEnv("WEBAUTHN_RP_ID", "localhost"),
		WebAuthnOrigins:           splitCSV(getEnv("WEBAUTHN_ORIGINS", "http://localhost:5173")),
		CORSAllowedOrigins:        splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "")),
		PasswordRecoveryEnabled:   strings.EqualFold(getEnv("PASSWORD_RECOVERY_ENABLED", "false"), "true"),
		PasswordResetBaseURL:      getEnv("PASSWORD_RESET_BASE_URL", "http://localhost:5173/"),
		SMTPHost:                  getEnv("SMTP_HOST", ""),
		SMTPPort:                  getEnv("SMTP_PORT", "465"),
		SMTPUser:                  getEnv("SMTP_USER", ""),
		SMTPPassword:              getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:                  getEnv("SMTP_FROM", ""),
		SMTPServerName:            getEnv("SMTP_SERVER_NAME", ""),
		OTLPTraceEndpoint:         getEnv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", ""),
		AgentCACertFile:           getEnv("AGENT_CA_CERT_FILE", ""),
		AgentCAKeyFile:            getEnv("AGENT_CA_KEY_FILE", ""),
		AgentCertificateTTL:       agentCertTTL,
		AgentEnrollmentTTL:        agentEnrollmentTTL,
		AgentConfigSigningKeyFile: getEnv("AGENT_CONFIG_SIGNING_KEY_FILE", ""),
	}

	production := cfg.IsProduction()
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
	if cfg.AgentConfigSigningKeyFile != "" && cfg.AgentCACertFile == "" {
		log.Fatal("FATAL: AGENT_CONFIG_SIGNING_KEY_FILE requires agent mTLS enrollment")
	}
	if production && cfg.AgentConfigSigningKeyFile == "" {
		log.Println("WARNING: secure remote agent configuration is disabled because AGENT_CONFIG_SIGNING_KEY_FILE is not configured")
	}

	return cfg
}

func (c Config) IsProduction() bool {
	return strings.EqualFold(c.Environment, "production") || strings.EqualFold(c.Environment, "prod")
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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

package config

import (
	"log"
	"os"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL       string
	RedisURL          string
	Port              string
	CelestrakGroups   []string
	JWTSecret         []byte
	JWTPreviousSecret []byte
	JWTAccessTTL      time.Duration
	TLSCertFile       string
	TLSKeyFile        string
	SeedAdminEmail    string
	SeedAdminPass     string
	Environment       string
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

	cfg := Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://lyrahcle:lyrahcle@localhost:5432/lyrahcle?sslmode=disable"),
		RedisURL:          getEnv("REDIS_URL", "localhost:6379"),
		Port:              getEnv("PORT", "8080"),
		CelestrakGroups:   strings.Split(getEnv("CELESTRAK_GROUPS", "stations"), ","),
		JWTSecret:         []byte(secret),
		JWTPreviousSecret: []byte(getEnv("JWT_SECRET_PREVIOUS", "")),
		JWTAccessTTL:      ttl,
		TLSCertFile:       getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:        getEnv("TLS_KEY_FILE", ""),
		SeedAdminEmail:    getEnv("SEED_ADMIN_EMAIL", "admin"),
		SeedAdminPass:     getEnv("SEED_ADMIN_PASSWORD", "admin"),
		Environment:       getEnv("ENV", "development"),
	}

	production := cfg.IsProduction()

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

	if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
		if production {
			log.Fatal("FATAL: ENV=production but TLS_CERT_FILE/TLS_KEY_FILE are not set. Refusing to serve plain HTTP in production.")
		}
		log.Println("WARNING: TLS_CERT_FILE/TLS_KEY_FILE not set; serving plain HTTP. Set both to enable HTTPS (see scripts/generate-dev-certs.sh).")
	}

	return cfg
}

func (c Config) IsProduction() bool {
	return strings.EqualFold(c.Environment, "production") || strings.EqualFold(c.Environment, "prod")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL     string
	RedisURL        string
	Port            string
	CelestrakGroups []string
	JWTSecret       []byte
	TLSCertFile     string
	TLSKeyFile      string
	SeedAdminEmail  string
	SeedAdminPass   string
}

func Load() Config {
	secret := getEnv("JWT_SECRET", "")
	insecureDefault := false
	if secret == "" {
		secret = "insecure-dev-secret-change-me"
		insecureDefault = true
	}

	cfg := Config{
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://lyrahcle:lyrahcle@localhost:5432/lyrahcle?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "localhost:6379"),
		Port:            getEnv("PORT", "8080"),
		CelestrakGroups: strings.Split(getEnv("CELESTRAK_GROUPS", "stations"), ","),
		JWTSecret:       []byte(secret),
		TLSCertFile:     getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:      getEnv("TLS_KEY_FILE", ""),
		SeedAdminEmail:  getEnv("SEED_ADMIN_EMAIL", "admin"),
		SeedAdminPass:   getEnv("SEED_ADMIN_PASSWORD", "admin"),
	}

	if insecureDefault {
		log.Println("WARNING: JWT_SECRET is not set; using an insecure default. Set JWT_SECRET in production.")
	}
	if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
		log.Println("WARNING: TLS_CERT_FILE/TLS_KEY_FILE not set; serving plain HTTP. Set both to enable HTTPS (see scripts/generate-dev-certs.sh).")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

package config

import (
	"os"
	"strings"
)

type Config struct {
	DatabaseURL     string
	RedisURL        string
	Port            string
	CelestrakGroups []string
}

func Load() Config {
	return Config{
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://lyrahcle:lyrahcle@localhost:5432/lyrahcle?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "localhost:6379"),
		Port:            getEnv("PORT", "8080"),
		CelestrakGroups: strings.Split(getEnv("CELESTRAK_GROUPS", "stations"), ","),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

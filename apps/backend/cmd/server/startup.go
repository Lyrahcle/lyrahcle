package main

import (
	"context"
	"log"

	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

func seedAdmin(ctx context.Context, users *store.UserStore, cfg config.Config) error {
	count, err := users.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := auth.HashPassword(cfg.SeedAdminPass)
	if err != nil {
		return err
	}

	seeded, err := users.CreateProvisioned(ctx, cfg.SeedAdminEmail, hash, string(auth.RoleAdmin))
	if err != nil {
		return err
	}
	if err := users.GrantPlatformAdministrator(ctx, seeded.ID); err != nil {
		return err
	}

	log.Println("============================================================")
	log.Printf("Created default admin account: %q / (seed password)", cfg.SeedAdminEmail)
	log.Println("This is a well-known default and MUST be changed immediately:")
	log.Println("  POST /api/v1/auth/change-password (after logging in)")
	log.Println("Override the seed via SEED_ADMIN_EMAIL / SEED_ADMIN_PASSWORD")
	log.Println("before first boot in any non-local environment.")
	log.Println("============================================================")

	return nil
}

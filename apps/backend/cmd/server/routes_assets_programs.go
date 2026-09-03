package main

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
	"net/http"
	"strings"
)

func (a *apiRoutes) registerAssetProgramRoutes(r chi.Router) {
	assets := a.assets
	auditLog := a.auditLog
	cfg := a.cfg
	isTokenRevoked := a.isTokenRevoked
	programs := a.programs
	rbac := a.rbac

	r.Route("/assets", func(r chi.Router) {
		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "assets.read", isTokenRevoked)).Get("/", func(w http.ResponseWriter, r *http.Request) {
			limit, offset := pageParams(r)
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			page, err := assets.ListByOrganization(r.Context(), orgID, limit, offset)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(page)
		})

		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "assets.manage", isTokenRevoked)).Post("/", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				ProgramID string `json:"program_id"`
				Kind      string `json:"kind"`
				Value     string `json:"value"`
				CPEURI    string `json:"cpe_uri"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			body.Value = strings.TrimSpace(body.Value)
			body.CPEURI = strings.TrimSpace(body.CPEURI)

			programID, err := uuid.Parse(body.ProgramID)
			if err != nil {
				http.Error(w, "program_id must be a valid UUID", http.StatusBadRequest)
				return
			}
			if err := validate.AssetKind(body.Kind); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := validate.AssetValue(body.Value); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := validate.CPEURI(body.CPEURI); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			if err := assets.UpsertInOrganizationWithCPE(r.Context(), orgID, programID, body.Kind, body.Value, body.CPEURI); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					http.Error(w, "program not found", http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if claims, ok := auth.FromContext(r.Context()); ok {
				logAuditInOrganization(r.Context(), auditLog, orgID, audit.ActorUser, claims.UserID.String(), audit.ActionAssetUpserted, clientIP(r), map[string]interface{}{
					"program_id": programID.String(), "kind": body.Kind, "value": body.Value,
				})
			}
			w.WriteHeader(http.StatusCreated)
		})

		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "assets.manage", isTokenRevoked)).Put("/{assetID}", func(w http.ResponseWriter, r *http.Request) {
			assetID, err := uuid.Parse(chi.URLParam(r, "assetID"))
			if err != nil {
				http.Error(w, "asset_id must be a valid UUID", http.StatusBadRequest)
				return
			}
			var body struct {
				ProgramID string `json:"program_id"`
				Kind      string `json:"kind"`
				Value     string `json:"value"`
				CPEURI    string `json:"cpe_uri"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			body.Value = strings.TrimSpace(body.Value)
			body.CPEURI = strings.TrimSpace(body.CPEURI)
			programID, err := uuid.Parse(body.ProgramID)
			if err != nil {
				http.Error(w, "program_id must be a valid UUID", http.StatusBadRequest)
				return
			}
			if err := validate.AssetKind(body.Kind); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := validate.AssetValue(body.Value); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := validate.CPEURI(body.CPEURI); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			asset, err := assets.UpdateInOrganizationWithCPE(r.Context(), orgID, assetID, programID, body.Kind, body.Value, body.CPEURI)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					http.Error(w, "asset not found", http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if claims, ok := auth.FromContext(r.Context()); ok {
				logAuditInOrganization(r.Context(), auditLog, orgID, audit.ActorUser, claims.UserID.String(), audit.ActionAssetUpdated, clientIP(r), map[string]interface{}{
					"asset_id": asset.ID.String(), "program_id": asset.ProgramID.String(), "kind": asset.Kind, "value": asset.Value,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(asset)
		})

		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "assets.manage", isTokenRevoked)).Delete("/{assetID}", func(w http.ResponseWriter, r *http.Request) {
			assetID, err := uuid.Parse(chi.URLParam(r, "assetID"))
			if err != nil {
				http.Error(w, "asset_id must be a valid UUID", http.StatusBadRequest)
				return
			}
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			if err := assets.DeleteInOrganization(r.Context(), orgID, assetID); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					http.Error(w, "asset not found", http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if claims, ok := auth.FromContext(r.Context()); ok {
				logAuditInOrganization(r.Context(), auditLog, orgID, audit.ActorUser, claims.UserID.String(), audit.ActionAssetDeleted, clientIP(r), map[string]interface{}{"asset_id": assetID.String()})
			}
			w.WriteHeader(http.StatusNoContent)
		})
	})

	r.Route("/programs", func(r chi.Router) {
		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "programs.read", isTokenRevoked)).Get("/", func(w http.ResponseWriter, r *http.Request) {
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			list, err := programs.ListByOrganization(r.Context(), orgID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)
		})

		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "programs.manage", isTokenRevoked)).Post("/", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Name string `json:"name"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			body.Name = strings.TrimSpace(body.Name)
			if err := validate.ProgramName(body.Name); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			program, err := programs.CreateInOrganization(r.Context(), orgID, body.Name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if claims, ok := auth.FromContext(r.Context()); ok {
				logAuditInOrganization(r.Context(), auditLog, orgID, audit.ActorUser, claims.UserID.String(), audit.ActionProgramCreated, clientIP(r), map[string]interface{}{
					"program_id": program.ID.String(), "name": program.Name,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(program)
		})

		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "programs.manage", isTokenRevoked)).Put("/{programID}", func(w http.ResponseWriter, r *http.Request) {
			programID, err := uuid.Parse(chi.URLParam(r, "programID"))
			if err != nil {
				http.Error(w, "program_id must be a valid UUID", http.StatusBadRequest)
				return
			}
			var body struct {
				Name string `json:"name"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			body.Name = strings.TrimSpace(body.Name)
			if err := validate.ProgramName(body.Name); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			program, err := programs.UpdateInOrganization(r.Context(), orgID, programID, body.Name)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					http.Error(w, "program not found", http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if claims, ok := auth.FromContext(r.Context()); ok {
				logAuditInOrganization(r.Context(), auditLog, orgID, audit.ActorUser, claims.UserID.String(), audit.ActionProgramUpdated, clientIP(r), map[string]interface{}{
					"program_id": program.ID.String(), "name": program.Name,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(program)
		})

		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "programs.manage", isTokenRevoked)).Delete("/{programID}", func(w http.ResponseWriter, r *http.Request) {
			programID, err := uuid.Parse(chi.URLParam(r, "programID"))
			if err != nil {
				http.Error(w, "program_id must be a valid UUID", http.StatusBadRequest)
				return
			}
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			if err := programs.DeleteInOrganization(r.Context(), orgID, programID); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					http.Error(w, "program not found", http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if claims, ok := auth.FromContext(r.Context()); ok {
				logAuditInOrganization(r.Context(), auditLog, orgID, audit.ActorUser, claims.UserID.String(), audit.ActionProgramDeleted, clientIP(r), map[string]interface{}{"program_id": programID.String()})
			}
			w.WriteHeader(http.StatusNoContent)
		})
	})
}

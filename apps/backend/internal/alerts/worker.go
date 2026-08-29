package alerts

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

// QueueKey is the Redis list cve-sync LPUSHes a cve_id onto after every
// successful upsert (new or updated CVE). This backend BLPOPs from it.
const QueueKey = "lyrahcle:cve-events"

type Worker struct {
	redis     *redis.Client
	spaceCVEs *store.SpaceCVEStore
	assets    *store.AssetStore
	alerts    *store.AssetAlertStore
	auditLog  *store.AuditStore
}

func NewWorker(redisClient *redis.Client, spaceCVEs *store.SpaceCVEStore, assets *store.AssetStore, alertStore *store.AssetAlertStore, auditLog *store.AuditStore) *Worker {
	return &Worker{redis: redisClient, spaceCVEs: spaceCVEs, assets: assets, alerts: alertStore, auditLog: auditLog}
}

// Run blocks, consuming cve_id events from QueueKey until ctx is canceled.
// Safe to call unconditionally in a goroutine: a nil Worker or one with no
// Redis client just returns immediately, so CVE-to-asset alerting is
// simply disabled rather than crashing when Redis isn't configured — the
// same graceful-degradation approach as the backend's other Redis-backed
// features (see internal/ratelimit, internal/tokenblocklist, internal/respcache).
func (w *Worker) Run(ctx context.Context) {
	if w == nil || w.redis == nil {
		return
	}
	log.Printf("alerts: worker started, consuming %s", QueueKey)

	for {
		if ctx.Err() != nil {
			log.Println("alerts: worker stopping")
			return
		}

		result, err := w.redis.BLPop(ctx, 5*time.Second, QueueKey).Result()
		if err == redis.Nil {
			continue // nothing queued within the timeout — loop and recheck ctx
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("alerts: blpop error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		cveID := result[1]
		if err := w.ProcessCVE(ctx, cveID); err != nil {
			log.Printf("alerts: process %s: %v", cveID, err)
		}
	}
}

// ProcessCVE matches one CVE against tracked assets by kind (see
// AssetKindsForKeyword) and records any new matches as alerts, each with
// an audit log entry. Idempotent: re-processing the same CVE ID against
// unchanged assets creates no duplicate alerts (store.AssetAlertStore.Create
// is a no-op on an existing asset/CVE pair).
//
// Exported and separate from Run's queue-draining loop so it can be
// exercised directly in tests against a real database, without needing a
// live Redis queue to feed it.
func (w *Worker) ProcessCVE(ctx context.Context, cveID string) error {
	cve, err := w.spaceCVEs.GetByID(ctx, cveID)
	if err != nil {
		return fmt.Errorf("lookup cve %s: %w", cveID, err)
	}

	kinds := AssetKindsForKeyword(cve.Keyword)
	if len(kinds) == 0 {
		return nil
	}

	matched, err := w.assets.ListByKinds(ctx, kinds)
	if err != nil {
		return fmt.Errorf("list assets for kinds %v: %w", kinds, err)
	}

	for _, a := range matched {
		created, err := w.alerts.Create(ctx, a.ID, cve.CVEID, cve.Keyword, cve.Severity)
		if err != nil {
			log.Printf("alerts: create alert for asset %s / cve %s: %v", a.ID, cve.CVEID, err)
			continue
		}
		if !created {
			continue // already alerted on this asset/CVE pair
		}

		if w.auditLog != nil {
			details := map[string]interface{}{
				"cve_id":      cve.CVEID,
				"asset_kind":  a.Kind,
				"asset_value": a.Value,
				"keyword":     cve.Keyword,
				"severity":    cve.Severity,
			}
			if err := w.auditLog.Insert(ctx, audit.ActorSystem, "cve-alert-matcher", audit.ActionAssetAlertCreated, "", details); err != nil {
				log.Printf("alerts: failed to record audit entry for asset %s / cve %s: %v", a.ID, cve.CVEID, err)
			}
		}
	}

	return nil
}

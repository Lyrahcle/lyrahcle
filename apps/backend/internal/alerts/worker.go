package alerts

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/metrics"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

const (
	// QueueKey is the pre-v13.9 Redis list. Workers keep draining it during a
	// rolling upgrade so events produced by an older cve-sync are not stranded.
	QueueKey = "lyrahcle:cve-events"

	StreamKey     = "lyrahcle:cve-events:v2"
	ConsumerGroup = "asset-alert-workers"
)

const (
	defaultClaimIdle = 30 * time.Second
	defaultReadBlock = 5 * time.Second
)

var migrateLegacyEventScript = redis.NewScript(`
local cve = redis.call("RPOP", KEYS[1])
if not cve then
  return nil
end
local id = redis.call("XADD", KEYS[2], "*", "cve_id", cve, "source", "legacy-list")
return {cve, id}
`)

type Worker struct {
	redis       *redis.Client
	spaceCVEs   *store.SpaceCVEStore
	assets      *store.AssetStore
	alerts      *store.AssetAlertStore
	auditLog    *store.AuditStore
	consumer    string
	claimIdle   time.Duration
	readBlock   time.Duration
	legacyBatch int64
	process     func(context.Context, string) error
}

func NewWorker(redisClient *redis.Client, spaceCVEs *store.SpaceCVEStore, assets *store.AssetStore, alertStore *store.AssetAlertStore, auditLog *store.AuditStore) *Worker {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "backend"
	}
	return &Worker{
		redis:       redisClient,
		spaceCVEs:   spaceCVEs,
		assets:      assets,
		alerts:      alertStore,
		auditLog:    auditLog,
		consumer:    hostname + "-" + uuid.NewString(),
		claimIdle:   defaultClaimIdle,
		readBlock:   defaultReadBlock,
		legacyBatch: 100,
	}
}

func (w *Worker) Run(ctx context.Context) {
	if w == nil || w.redis == nil {
		return
	}
	if err := w.ensureConsumerGroup(ctx); err != nil {
		log.Printf("alerts: ensure stream consumer group: %v", err)
		return
	}
	log.Printf("alerts: worker started, consuming %s as %s/%s", StreamKey, ConsumerGroup, w.consumer)

	for {
		if ctx.Err() != nil {
			log.Println("alerts: worker stopping")
			return
		}

		if err := w.drainLegacyQueue(ctx); err != nil && ctx.Err() == nil {
			log.Printf("alerts: legacy queue migration failed: %v", err)
		}

		if handled, err := w.reclaimOne(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("alerts: pending message reclaim failed: %v", err)
			time.Sleep(time.Second)
			continue
		} else if handled {
			continue
		}

		streams, err := w.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    ConsumerGroup,
			Consumer: w.consumer,
			Streams:  []string{StreamKey, ">"},
			Count:    1,
			Block:    w.readBlock,
		}).Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("alerts: xreadgroup error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				w.handleMessage(ctx, message, false)
			}
		}
	}
}

func (w *Worker) ensureConsumerGroup(ctx context.Context) error {
	err := w.redis.XGroupCreateMkStream(ctx, StreamKey, ConsumerGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

// drainLegacyQueue keeps rolling upgrades lossless. Old cve-sync versions use
// LPUSH on QueueKey; each legacy item is atomically popped and re-published to
// the durable stream. Duplicate deliveries are harmless because PostgreSQL
// enforces asset/CVE idempotency.
func (w *Worker) drainLegacyQueue(ctx context.Context) error {
	for i := int64(0); i < w.legacyBatch; i++ {
		result, err := migrateLegacyEventScript.Run(ctx, w.redis, []string{QueueKey, StreamKey}).Slice()
		if err == redis.Nil {
			return nil
		}
		if err != nil {
			return err
		}
		if len(result) == 0 {
			return nil
		}
		metrics.BackgroundJobEventsTotal.WithLabelValues("cve-alerts", "legacy_migrated").Inc()
	}
	return nil
}

func (w *Worker) reclaimOne(ctx context.Context) (bool, error) {
	messages, _, err := w.redis.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   StreamKey,
		Group:    ConsumerGroup,
		Consumer: w.consumer,
		MinIdle:  w.claimIdle,
		Start:    "0-0",
		Count:    1,
	}).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if len(messages) == 0 {
		return false, nil
	}
	w.handleMessage(ctx, messages[0], true)
	return true, nil
}

func (w *Worker) handleMessage(ctx context.Context, message redis.XMessage, reclaimed bool) {
	cveID, _ := message.Values["cve_id"].(string)
	if strings.TrimSpace(cveID) == "" {
		log.Printf("alerts: dropping malformed stream message %s without cve_id", message.ID)
		metrics.BackgroundJobEventsTotal.WithLabelValues("cve-alerts", "malformed").Inc()
		w.ackMessage(ctx, message.ID)
		return
	}

	if reclaimed {
		metrics.BackgroundJobEventsTotal.WithLabelValues("cve-alerts", "reclaimed").Inc()
	}
	process := w.ProcessCVE
	if w.process != nil {
		process = w.process
	}
	if err := process(ctx, cveID); err != nil {
		// Deliberately do not ACK. The message remains pending and becomes
		// claimable after claimIdle, including after a process crash/restart.
		metrics.BackgroundJobEventsTotal.WithLabelValues("cve-alerts", "failed").Inc()
		log.Printf("alerts: process %s (message %s): %v", cveID, message.ID, err)
		return
	}
	metrics.BackgroundJobEventsTotal.WithLabelValues("cve-alerts", "processed").Inc()
	w.ackMessage(ctx, message.ID)
}

func (w *Worker) ackMessage(ctx context.Context, messageID string) {
	pipe := w.redis.TxPipeline()
	pipe.XAck(ctx, StreamKey, ConsumerGroup, messageID)
	pipe.XDel(ctx, StreamKey, messageID)
	if _, err := pipe.Exec(ctx); err != nil {
		// Failure to ACK is safe: the event can be redelivered and the durable
		// stores are idempotent. Surface it so operators can see retry pressure.
		metrics.BackgroundJobEventsTotal.WithLabelValues("cve-alerts", "ack_failed").Inc()
		log.Printf("alerts: ack stream message %s: %v", messageID, err)
	}
}

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

	var processingErrors []error
	for _, a := range matched {
		_, err := w.alerts.Create(ctx, a.ID, cve.CVEID, cve.Keyword, cve.Severity)
		if err != nil {
			log.Printf("alerts: create alert for asset %s / cve %s: %v", a.ID, cve.CVEID, err)
			processingErrors = append(processingErrors, fmt.Errorf("create alert for asset %s: %w", a.ID, err))
			continue
		}

		if w.auditLog != nil {
			details := map[string]interface{}{
				"organization_id": a.OrganizationID.String(),
				"cve_id":          cve.CVEID,
				"asset_kind":      a.Kind,
				"asset_value":     a.Value,
				"keyword":         cve.Keyword,
				"severity":        cve.Severity,
			}
			dedupeKey := "asset-alert:" + a.ID.String() + ":" + cve.CVEID
			if _, err := w.auditLog.InsertIdempotentInOrganization(ctx, a.OrganizationID, dedupeKey, audit.ActorSystem, "cve-alert-matcher", audit.ActionAssetAlertCreated, "", details); err != nil {
				log.Printf("alerts: failed to record audit entry for asset %s / cve %s: %v", a.ID, cve.CVEID, err)
				processingErrors = append(processingErrors, fmt.Errorf("audit alert for asset %s: %w", a.ID, err))
			}
		}
	}

	return errors.Join(processingErrors...)
}

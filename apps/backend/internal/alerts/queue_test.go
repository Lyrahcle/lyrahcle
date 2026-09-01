package alerts

import (
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func testQueueRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set, skipping Redis queue integration test")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		t.Fatalf("redis ping: %v", err)
	}
	if err := client.Del(ctx, QueueKey, StreamKey).Err(); err != nil {
		client.Close()
		t.Fatalf("clear queue keys: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Del(context.Background(), QueueKey, StreamKey).Err()
		_ = client.Close()
	})
	return client
}

func newQueueTestWorker(client *redis.Client, consumer string, process func(context.Context, string) error) *Worker {
	return &Worker{
		redis:       client,
		consumer:    consumer,
		claimIdle:   20 * time.Millisecond,
		readBlock:   20 * time.Millisecond,
		legacyBatch: 100,
		process:     process,
	}
}

func TestPendingEventIsReclaimedAfterWorkerCrash(t *testing.T) {
	client := testQueueRedis(t)
	ctx := context.Background()
	first := newQueueTestWorker(client, "first-"+uuid.NewString(), nil)
	if err := first.ensureConsumerGroup(ctx); err != nil {
		t.Fatalf("ensure group: %v", err)
	}
	messageID, err := client.XAdd(ctx, &redis.XAddArgs{Stream: StreamKey, Values: map[string]interface{}{"cve_id": "CVE-2026-90001"}}).Result()
	if err != nil {
		t.Fatalf("XAdd: %v", err)
	}
	streams, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: ConsumerGroup, Consumer: first.consumer, Streams: []string{StreamKey, ">"}, Count: 1,
	}).Result()
	if err != nil || len(streams) != 1 || len(streams[0].Messages) != 1 {
		t.Fatalf("claim message before simulated crash: streams=%v err=%v", streams, err)
	}
	if streams[0].Messages[0].ID != messageID {
		t.Fatalf("claimed ID=%s want %s", streams[0].Messages[0].ID, messageID)
	}
	time.Sleep(30 * time.Millisecond)

	var processed atomic.Int32
	second := newQueueTestWorker(client, "second-"+uuid.NewString(), func(_ context.Context, cveID string) error {
		if cveID != "CVE-2026-90001" {
			t.Fatalf("reclaimed cve_id=%q", cveID)
		}
		processed.Add(1)
		return nil
	})
	handled, err := second.reclaimOne(ctx)
	if err != nil || !handled {
		t.Fatalf("reclaimOne: handled=%v err=%v", handled, err)
	}
	if processed.Load() != 1 {
		t.Fatalf("processed=%d want 1", processed.Load())
	}
	pending, err := client.XPending(ctx, StreamKey, ConsumerGroup).Result()
	if err != nil {
		t.Fatalf("XPending: %v", err)
	}
	if pending.Count != 0 {
		t.Fatalf("pending count=%d want 0 after ACK", pending.Count)
	}
	if n, err := client.XLen(ctx, StreamKey).Result(); err != nil || n != 0 {
		t.Fatalf("stream length=%d err=%v want 0 after ACK+DEL", n, err)
	}
}

func TestFailedEventStaysPendingForRetry(t *testing.T) {
	client := testQueueRedis(t)
	ctx := context.Background()
	worker := newQueueTestWorker(client, "retry-"+uuid.NewString(), func(context.Context, string) error {
		return errors.New("synthetic downstream failure")
	})
	if err := worker.ensureConsumerGroup(ctx); err != nil {
		t.Fatalf("ensure group: %v", err)
	}
	if _, err := client.XAdd(ctx, &redis.XAddArgs{Stream: StreamKey, Values: map[string]interface{}{"cve_id": "CVE-2026-90002"}}).Result(); err != nil {
		t.Fatalf("XAdd: %v", err)
	}
	streams, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: ConsumerGroup, Consumer: worker.consumer, Streams: []string{StreamKey, ">"}, Count: 1,
	}).Result()
	if err != nil || len(streams) != 1 || len(streams[0].Messages) != 1 {
		t.Fatalf("XReadGroup: streams=%v err=%v", streams, err)
	}
	worker.handleMessage(ctx, streams[0].Messages[0], false)
	pending, err := client.XPending(ctx, StreamKey, ConsumerGroup).Result()
	if err != nil {
		t.Fatalf("XPending: %v", err)
	}
	if pending.Count != 1 {
		t.Fatalf("pending count=%d want 1 after failed processing", pending.Count)
	}
}

func TestLegacyListIsMigratedIntoDurableStream(t *testing.T) {
	client := testQueueRedis(t)
	ctx := context.Background()
	worker := newQueueTestWorker(client, "migration-"+uuid.NewString(), nil)
	if err := worker.ensureConsumerGroup(ctx); err != nil {
		t.Fatalf("ensure group: %v", err)
	}
	if err := client.LPush(ctx, QueueKey, "CVE-2026-90003").Err(); err != nil {
		t.Fatalf("LPush legacy: %v", err)
	}
	if err := worker.drainLegacyQueue(ctx); err != nil {
		t.Fatalf("drainLegacyQueue: %v", err)
	}
	if n, err := client.LLen(ctx, QueueKey).Result(); err != nil || n != 0 {
		t.Fatalf("legacy queue length=%d err=%v want 0", n, err)
	}
	if n, err := client.XLen(ctx, StreamKey).Result(); err != nil || n != 1 {
		t.Fatalf("stream length=%d err=%v want 1", n, err)
	}
}

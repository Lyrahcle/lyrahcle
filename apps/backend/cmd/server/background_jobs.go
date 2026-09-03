package main

import (
	"context"
	"log"
	"time"

	"github.com/mlko01/lyrahcle/apps/backend/internal/joblock"
	"github.com/mlko01/lyrahcle/apps/backend/internal/metrics"
)

func runSingletonScheduledJob(ctx context.Context, locker *joblock.Locker, name string, ttl time.Duration, job func()) {
	unlock, acquired, err := locker.Try(ctx, name, ttl)
	if err != nil {
		metrics.ScheduledJobLeadershipTotal.WithLabelValues(name, "error").Inc()
		log.Printf("scheduler: distributed lock %q: %v", name, err)
		return
	}
	if !acquired {
		metrics.ScheduledJobLeadershipTotal.WithLabelValues(name, "skipped").Inc()
		log.Printf("scheduler: skipping job %q because another backend replica owns the lease", name)
		return
	}
	metrics.ScheduledJobLeadershipTotal.WithLabelValues(name, "acquired").Inc()
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := unlock(releaseCtx); err != nil {
			metrics.ScheduledJobLeadershipTotal.WithLabelValues(name, "release_error").Inc()
			log.Printf("scheduler: release distributed lock %q: %v", name, err)
		}
	}()
	job()
}

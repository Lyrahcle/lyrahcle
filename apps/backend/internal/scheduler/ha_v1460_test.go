package scheduler

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestEveryValidatesInputs(t *testing.T) {
	var nilScheduler *Scheduler
	if err := nilScheduler.Every("* * * * *", "job", func() {}); err == nil {
		t.Fatal("nil scheduler must fail")
	}

	s := New()
	for _, tc := range []struct {
		spec string
		name string
		job  func()
	}{
		{"* * * * *", "", func() {}},
		{"* * * * *", "   ", func() {}},
		{"* * * * *", "job", nil},
		{"not-a-cron", "job", func() {}},
	} {
		if err := s.Every(tc.spec, tc.name, tc.job); err == nil {
			t.Fatalf("Every(%q,%q) unexpectedly succeeded", tc.spec, tc.name)
		}
	}
}

func TestSchedulerRecoversPanicsFromJobs(t *testing.T) {
	s := New()
	if err := s.Every("* * * * *", "panic-job", func() { panic("boom") }); err != nil {
		t.Fatal(err)
	}
	entries := s.cron.Entries()
	if len(entries) != 1 {
		t.Fatalf("entries=%d want 1", len(entries))
	}

	// The configured cron recovery chain must contain the panic rather than
	// allowing it to escape the scheduler goroutine/process.
	entries[0].Job.Run()
}

func TestSchedulerSkipsOverlappingRunsInSameReplica(t *testing.T) {
	s := New()
	started := make(chan struct{})
	release := make(chan struct{})
	var runs atomic.Int32

	if err := s.Every("* * * * *", "slow-job", func() {
		if runs.Add(1) == 1 {
			close(started)
		}
		<-release
	}); err != nil {
		t.Fatal(err)
	}
	entry := s.cron.Entries()[0]

	done := make(chan struct{})
	go func() {
		entry.Job.Run()
		close(done)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first run did not start")
	}

	// This returns immediately because SkipIfStillRunning owns the wrapped job.
	entry.Job.Run()
	if got := runs.Load(); got != 1 {
		t.Fatalf("overlapping executions=%d want 1", got)
	}

	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("first run did not finish")
	}
}

func TestStartAndStopAreNilSafe(t *testing.T) {
	var nilScheduler *Scheduler
	nilScheduler.Start()
	nilScheduler.Stop()

	s := New()
	s.Start()
	s.Stop()
}

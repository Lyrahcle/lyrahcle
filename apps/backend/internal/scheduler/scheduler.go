package scheduler

import (
	"log"

	"github.com/robfig/cron/v3"
)

// Scheduler wraps robfig/cron for periodic monitoring tasks
// (recon sweeps, TLE refresh, CVE sync, etc.)
type Scheduler struct {
	cron *cron.Cron
}

func New() *Scheduler {
	return &Scheduler{cron: cron.New()}
}

// Every registers a job with a standard cron spec, e.g. "0 */6 * * *" (every 6h).
func (s *Scheduler) Every(spec string, name string, job func()) error {
	_, err := s.cron.AddFunc(spec, func() {
		log.Printf("scheduler: running job %q", name)
		job()
	})
	return err
}

func (s *Scheduler) Start() { s.cron.Start() }
func (s *Scheduler) Stop()  { s.cron.Stop() }

package scheduler

import (
	"log"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron *cron.Cron
}

func New() *Scheduler {
	return &Scheduler{cron: cron.New()}
}

func (s *Scheduler) Every(spec string, name string, job func()) error {
	_, err := s.cron.AddFunc(spec, func() {
		log.Printf("scheduler: running job %q", name)
		job()
	})
	return err
}

func (s *Scheduler) Start() { s.cron.Start() }
func (s *Scheduler) Stop()  { s.cron.Stop() }

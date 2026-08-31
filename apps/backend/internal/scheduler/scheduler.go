package scheduler

import (
	"errors"
	"log"
	"strings"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron *cron.Cron
}

func New() *Scheduler {
	return &Scheduler{cron: cron.New()}
}

func (s *Scheduler) Every(spec string, name string, job func()) error {
	if s == nil || s.cron == nil {
		return errors.New("scheduler: unavailable")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("scheduler: job name is required")
	}
	if job == nil {
		return errors.New("scheduler: job function is required")
	}
	wrapped := cron.NewChain(
		cron.Recover(cron.DefaultLogger),
		cron.SkipIfStillRunning(cron.DefaultLogger),
	).Then(cron.FuncJob(func() {
		log.Printf("scheduler: running job %q", name)
		job()
	}))
	_, err := s.cron.AddJob(spec, wrapped)
	return err
}

func (s *Scheduler) Start() {
	if s != nil && s.cron != nil {
		s.cron.Start()
	}
}

func (s *Scheduler) Stop() {
	if s == nil || s.cron == nil {
		return
	}
	<-s.cron.Stop().Done()
}

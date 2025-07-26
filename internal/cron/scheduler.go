package cron

import (
	"log"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/tanq16/nottif/internal/config"
	"github.com/tanq16/nottif/internal/notifier"
)

// AddEventFunc is a function type for adding an event to the log
type AddEventFunc func(source, message string, success bool)

// Scheduler manages the cron jobs
type Scheduler struct {
	scheduler  gocron.Scheduler
	config     *config.Config
	notifier   *notifier.Notifier
	addEvent   AddEventFunc
	jobEntries map[string]uuid.UUID
}

// NewScheduler creates and configures a new cron scheduler
func NewScheduler(cfg *config.Config, n *notifier.Notifier, addEvent AddEventFunc) (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}
	return &Scheduler{
		scheduler:  s,
		config:     cfg,
		notifier:   n,
		addEvent:   addEvent,
		jobEntries: make(map[string]uuid.UUID),
	}, nil
}

func (s *Scheduler) Start() {
	s.config.Mu.RLock()
	for _, job := range s.config.CronJobs {
		_ = s.AddJob(job)
	}
	s.config.Mu.RUnlock()
	s.scheduler.Start()
}

func (s *Scheduler) AddJob(job config.CronJob) error {
	task := func() {
		log.Printf("Running cron job: %s", job.Message)
		err := s.notifier.SendMessage(
			job.Message,
			"Nottif Cron",
			"",
		)
		s.addEvent("Cron", job.Message, err == nil)
	}
	newJob, err := s.scheduler.NewJob(
		gocron.CronJob(
			job.Schedule,
			false, // standard 5-field cron expression
		),
		gocron.NewTask(task),
	)
	if err != nil {
		log.Printf("Error adding cron job '%s' with schedule '%s': %v", job.Message, job.Schedule, err)
		return err
	}
	log.Printf("Scheduled cron job '%s' with schedule '%s'", job.Message, job.Schedule)
	s.jobEntries[job.ID] = newJob.ID()
	return nil
}

func (s *Scheduler) RemoveJob(jobID string) {
	gocronJobID, ok := s.jobEntries[jobID]
	if !ok {
		log.Printf("Attempted to remove a cron job with ID '%s' that was not found in the scheduler.", jobID)
		return
	}
	if err := s.scheduler.RemoveJob(gocronJobID); err != nil {
		log.Printf("Error removing job with ID '%s' from scheduler: %v", jobID, err)
	}
	delete(s.jobEntries, jobID)
	log.Printf("Removed cron job with ID '%s' from scheduler.", jobID)
}

package cron

import (
	"log"

	"github.com/robfig/cron/v3"
	"github.com/tanq16/nottif/internal/config"
	"github.com/tanq16/nottif/internal/notifier"
)

// AddEventFunc is a function type for adding an event to the log.
type AddEventFunc func(source, message string, success bool)

// Scheduler manages the cron jobs.
type Scheduler struct {
	cron       *cron.Cron
	config     *config.Config
	notifier   *notifier.Notifier
	addEvent   AddEventFunc
	jobEntries map[string]cron.EntryID // Maps our job ID to the cron library's EntryID
}

// NewScheduler creates and configures a new cron scheduler.
func NewScheduler(cfg *config.Config, n *notifier.Notifier, addEvent AddEventFunc) *Scheduler {
	return &Scheduler{
		cron:       cron.New(),
		config:     cfg,
		notifier:   n,
		addEvent:   addEvent,
		jobEntries: make(map[string]cron.EntryID),
	}
}

// Start initializes jobs from config and starts the cron scheduler.
func (s *Scheduler) Start() {
	s.config.Mu.RLock()
	for _, job := range s.config.CronJobs {
		s.AddJob(job)
	}
	s.config.Mu.RUnlock()
	s.cron.Start()
}

// AddJob adds a new cron job to the scheduler.
func (s *Scheduler) AddJob(job config.CronJob) {
	entryID, err := s.cron.AddFunc(job.Schedule, func() {
		log.Printf("Running cron job: %s", job.Message)
		err := s.notifier.SendMessage(
			job.Message,
			"Nottif Cron", // Username for cron jobs
			"",            // Default avatar
		)
		s.addEvent("Cron", job.Message, err == nil)
	})

	if err != nil {
		log.Printf("Error adding cron job '%s' with schedule '%s': %v", job.Message, job.Schedule, err)
		return
	}

	log.Printf("Scheduled cron job '%s' with schedule '%s'", job.Message, job.Schedule)
	s.jobEntries[job.ID] = entryID
}

// RemoveJob removes a cron job from the scheduler.
func (s *Scheduler) RemoveJob(jobID string) {
	entryID, ok := s.jobEntries[jobID]
	if !ok {
		log.Printf("Attempted to remove a cron job with ID '%s' that was not found in the scheduler.", jobID)
		return
	}
	s.cron.Remove(entryID)
	delete(s.jobEntries, jobID)
	log.Printf("Removed cron job with ID '%s' from scheduler.", jobID)
}

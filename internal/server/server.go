package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/tanq16/nottif/internal/config"
	"github.com/tanq16/nottif/internal/cron"
	"github.com/tanq16/nottif/internal/notifier"
)

const maxEvents = 10

// Event represents a logged notification attempt.
type Event struct {
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"` // e.g., "API", "Cron", "Test"
	Message   string    `json:"message"`
	Success   bool      `json:"success"`
}

// Server holds the dependencies for the HTTP server.
type Server struct {
	Router    *mux.Router
	Config    *config.Config
	Notifier  *notifier.Notifier
	Scheduler *cron.Scheduler
	events    []Event
	eventsMu  sync.RWMutex
}

// New creates and initializes a new Server instance.
func New(cfg *config.Config, frontendFS embed.FS) (*Server, error) {
	n := notifier.New(cfg.WebhookURL)

	s := &Server{
		Router:   mux.NewRouter(),
		Config:   cfg,
		Notifier: n,
		events:   make([]Event, 0, maxEvents),
	}

	// Pass the AddEvent method to the scheduler
	s.Scheduler = cron.NewScheduler(cfg, n, s.addEvent)

	s.routes(frontendFS)
	return s, nil
}

// addEvent adds a new event to the in-memory log, ensuring the log does not exceed its max size.
func (s *Server) addEvent(source, message string, success bool) {
	s.eventsMu.Lock()
	defer s.eventsMu.Unlock()

	event := Event{
		Timestamp: time.Now(),
		Source:    source,
		Message:   message,
		Success:   success,
	}

	// Prepend the new event
	s.events = append([]Event{event}, s.events...)

	// Trim the slice if it exceeds the max size
	if len(s.events) > maxEvents {
		s.events = s.events[:maxEvents]
	}
}

// routes sets up all the application routes.
func (s *Server) routes(frontendFS embed.FS) {
	api := s.Router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/healthcheck", s.handleHealthCheck()).Methods("GET")
	api.HandleFunc("/webhook/test", s.handleTestWebhook()).Methods("POST")
	api.HandleFunc("/webhook/update", s.handleUpdateWebhook()).Methods("POST")
	api.HandleFunc("/cron/list", s.handleListCrons()).Methods("GET")
	api.HandleFunc("/cron/add", s.handleAddCron()).Methods("POST")
	api.HandleFunc("/cron/delete/{id}", s.handleDeleteCron()).Methods("DELETE")
	api.HandleFunc("/send", s.handleSendNotification()).Methods("POST")
	api.HandleFunc("/events", s.handleGetEvents()).Methods("GET")

	strippedFS, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		log.Fatalf("Failed to strip frontend prefix: %v", err)
	}
	fileServer := http.FileServer(http.FS(strippedFS))

	s.Router.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := r.URL.Path; path.Ext(p) == "" {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	}))
}

func (s *Server) handleHealthCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}

func (s *Server) handleGetEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.eventsMu.RLock()
		defer s.eventsMu.RUnlock()
		respondWithJSON(w, http.StatusOK, s.events)
	}
}

func (s *Server) handleListCrons() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.Config.Mu.RLock()
		defer s.Config.Mu.RUnlock()
		respondWithJSON(w, http.StatusOK, s.Config.CronJobs)
	}
}

func (s *Server) handleAddCron() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var job config.CronJob
		if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		job.ID = uuid.New().String()

		s.Scheduler.AddJob(job)

		s.Config.Mu.Lock()
		s.Config.CronJobs = append(s.Config.CronJobs, job)
		err := s.Config.Save()
		s.Config.Mu.Unlock()

		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to save config")
			return
		}

		s.addEvent("System", fmt.Sprintf("Added cron job: %s", job.Message), true)
		respondWithJSON(w, http.StatusCreated, job)
	}
}

func (s *Server) handleDeleteCron() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		s.Config.Mu.Lock()
		defer s.Config.Mu.Unlock()

		var jobToDelete config.CronJob
		found := false
		var updatedJobs []config.CronJob
		for _, job := range s.Config.CronJobs {
			if job.ID == id {
				found = true
				jobToDelete = job
				continue
			}
			updatedJobs = append(updatedJobs, job)
		}

		if !found {
			respondWithError(w, http.StatusNotFound, "Cron job not found")
			return
		}

		s.Scheduler.RemoveJob(id)

		s.Config.CronJobs = updatedJobs
		if err := s.Config.Save(); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to save config")
			return
		}

		s.addEvent("System", fmt.Sprintf("Deleted cron job: %s", jobToDelete.Message), true)
		respondWithJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func (s *Server) handleUpdateWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		s.Config.Mu.Lock()
		s.Config.WebhookURL = payload.URL
		s.Notifier.SetWebhookURL(payload.URL)
		err := s.Config.Save()
		s.Config.Mu.Unlock()

		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to save config")
			return
		}
		s.addEvent("System", "Updated Webhook URL", true)
		respondWithJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

func (s *Server) handleTestWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		message := "This is a test notification from Nottif!"
		err := s.Notifier.SendMessage(message, "Nottif Test", "")
		s.addEvent("Test", "Test Notification", err == nil)
		if err != nil {
			log.Printf("Failed to send test notification: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to send test notification")
			return
		}
		respondWithJSON(w, http.StatusOK, map[string]string{"status": "sent"})
	}
}

func (s *Server) handleSendNotification() http.HandlerFunc {
	type requestPayload struct {
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
		Content   string `json:"content"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var payload requestPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}
		if payload.Content == "" {
			respondWithError(w, http.StatusBadRequest, "Content field is required")
			return
		}

		err := s.Notifier.SendMessage(payload.Content, payload.Username, payload.AvatarURL)
		s.addEvent("API", payload.Content, err == nil)
		if err != nil {
			log.Printf("Failed to send notification via API: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to send notification")
			return
		}
		respondWithJSON(w, http.StatusOK, map[string]string{"status": "sent"})
	}
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

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

// Event is a logged notification attempt
type Event struct {
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
	Success   bool      `json:"success"`
}

// Broadcaster manages active SSE client connections
type Broadcaster struct {
	clients   map[chan []byte]bool
	new       chan (chan []byte)
	closing   chan (chan []byte)
	broadcast chan []byte
	mu        sync.Mutex
}

type Server struct {
	Router      *mux.Router
	Config      *config.Config
	Notifier    *notifier.Notifier
	Scheduler   *cron.Scheduler
	events      []Event
	eventsMu    sync.RWMutex
	Broadcaster *Broadcaster
}

func NewBroadcaster() *Broadcaster {
	b := &Broadcaster{
		clients:   make(map[chan []byte]bool),
		new:       make(chan (chan []byte)),
		closing:   make(chan (chan []byte)),
		broadcast: make(chan []byte),
	}
	go b.listen()
	return b
}

func (b *Broadcaster) listen() {
	for {
		select {
		case client := <-b.new:
			b.mu.Lock()
			b.clients[client] = true
			b.mu.Unlock()
			log.Println("SSE client connected.")
		case client := <-b.closing:
			b.mu.Lock()
			delete(b.clients, client)
			close(client)
			b.mu.Unlock()
			log.Println("SSE client disconnected.")
		case msg := <-b.broadcast:
			b.mu.Lock()
			for client := range b.clients {
				select {
				case client <- msg:
				default:
					delete(b.clients, client)
					close(client)
				}
			}
			b.mu.Unlock()
		}
	}
}

// New creates and initializes a new Server instance
func New(cfg *config.Config, frontendFS embed.FS) (*Server, error) {
	n := notifier.New(cfg.WebhookURL)
	s := &Server{
		Router:      mux.NewRouter(),
		Config:      cfg,
		Notifier:    n,
		events:      make([]Event, 0, maxEvents),
		Broadcaster: NewBroadcaster(),
	}
	scheduler, err := cron.NewScheduler(cfg, n, s.addEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to create cron scheduler: %w", err)
	}
	s.Scheduler = scheduler
	s.routes(frontendFS)
	return s, nil
}

// addEvent adds a new event to the in-memory log and broadcasts the full list
func (s *Server) addEvent(source, message string, success bool) {
	s.eventsMu.Lock()
	defer s.eventsMu.Unlock()
	displayMessage := message
	if source != "API" && len(displayMessage) > 25 {
		displayMessage = displayMessage[:25] + "..."
	}
	event := Event{
		Timestamp: time.Now(),
		Source:    source,
		Message:   displayMessage,
		Success:   success,
	}
	s.events = append([]Event{event}, s.events...) // prepend for ordering
	if len(s.events) > maxEvents {
		s.events = s.events[:maxEvents]
	}
	eventsJSON, err := json.Marshal(s.events)
	if err == nil {
		s.Broadcaster.broadcast <- eventsJSON
	} else {
		log.Printf("Error marshaling events for SSE broadcast: %v", err)
	}
}

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
	api.HandleFunc("/events/stream", s.handleStreamEvents()).Methods("GET")
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

// handleStreamEvents handles the SSE connection.
func (s *Server) handleStreamEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			respondWithError(w, http.StatusInternalServerError, "Streaming unsupported")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		clientChan := make(chan []byte)
		s.Broadcaster.new <- clientChan
		defer func() {
			s.Broadcaster.closing <- clientChan
		}()
		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, open := <-clientChan:
				if !open {
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", msg)
				flusher.Flush()
			}
		}
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
		if err := s.Scheduler.AddJob(job); err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.Config.Mu.Lock()
		s.Config.CronJobs = append(s.Config.CronJobs, job)
		err := s.Config.Save()
		s.Config.Mu.Unlock()
		if err != nil {
			// If saving fails, try to roll back by removing the job from the scheduler
			s.Scheduler.RemoveJob(job.ID)
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

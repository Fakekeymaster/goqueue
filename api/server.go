package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Fakekeymaster/goqueue/config"
	"github.com/Fakekeymaster/goqueue/queue"
	"github.com/Fakekeymaster/goqueue/store"
	"github.com/google/uuid"
)

// Server holds everything the API needs to handle requests.
// It wraps Go's http.Server and adds our app dependencies.
type Server struct {
	cfg   *config.Config
	store *store.Store
	http  *http.Server
}

// NewServer wires up all routes and returns a ready-to-start server.
// All configuration happens here — not scattered across the codebase.
func NewServer(cfg *config.Config, s *store.Store) *Server {
	srv := &Server{cfg: cfg, store: s}

	mux := http.NewServeMux()

	// Route registration — pattern → handler method
	// Note: "/jobs" matches exactly /jobs
	//       "/jobs/" matches /jobs/ AND /jobs/anything
	// This is how we handle both list and get-by-id with one mux.
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/jobs", srv.handleJobs)
	mux.HandleFunc("/jobs/", srv.handleJobByID)
	mux.HandleFunc("/stats", srv.handleStats)

	srv.http = &http.Server{
		Addr:    ":" + cfg.APIPort,
		Handler: loggingMiddleware(mux), // wrap mux with logging

		// Setting timeouts so that slow clients can't
		// hold connections open forever and exhaust our server.
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return srv
}

//Start func starts the server
//it is a blocking call so run it in a goroutine
func (s *Server) Start() error {
	log.Printf("[api] listening on: %s", s.cfg.APIPort)
	return s.http.ListenAndServe()
}

// Shutdown gracefully stops the server.
// In-flight requests are allowed to complete within the context deadline.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

// handleHealth is the simplest possible endpoint.
// Load balancers ping this to know if the server is alive.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// One function, two behaviours — determined by r.Method.
func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.submitJob(w, r)
	case http.MethodGet:
		s.listJobs(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleJobByID serves GET /jobs/{id}
func (s *Server) handleJobByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Strip the "/jobs/" prefix to get just the ID.
	// e.g. "/jobs/abc-123" → "abc-123"
	id := strings.TrimPrefix(r.URL.Path, "/jobs/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing job id")
		return
	}

	job, err := s.store.GetJob(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, job)
}

// handleStats serves GET /stats
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	stats, err := s.store.GetStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// submitRequest is what the client sends in the POST /jobs body.
// Using a dedicated struct (not map[string]any) gives us
// type safety and self-documentation.
type submitRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Priority string `json:"priority"`
}

func (s *Server) submitJob(w http.ResponseWriter, r *http.Request) {
	var req submitRequest

	// Decode the JSON body into our struct.
	// If the body is malformed JSON, this returns an error.
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Name == "" || req.Type == "" {
		writeError(w, http.StatusBadRequest, "name and type are required")
		return
	}

	job := &queue.Job{
		ID:         uuid.New().String(),
		Name:       req.Name,
		Type:       req.Type,
		Priority:   queue.ParsePriority(req.Priority),
		Status:     queue.StatusPending,
		MaxRetries: s.cfg.MaxRetries,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.store.Enqueue(r.Context(), job); err != nil {
		writeError(w, http.StatusInternalServerError,
			fmt.Sprintf("enqueue failed: %v", err))
		return
	}

	// 201 Created — not 200 OK.
	// 201 specifically means "a new resource was created."
	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.store.ListJobs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

// ── Helpers ────────────────────────────────────────────────

// writeJSON serializes v to JSON and writes it with the given status code.
// Setting Content-Type before WriteHeader is important —
// headers must always be set before the body is written.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// writeError is a convenience wrapper for error responses.
func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// loggingMiddleware wraps any handler and logs every request.
// This is the middleware pattern — it runs before and after the handler.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r) // call the actual handler
		log.Printf("[api] %s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
package intelserver

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/rs/zerolog/log"
)

type Server struct {
	listenAddr string
	store      *Store
}

func NewServer(listenAddr string) (*Server, error) {
	store, err := NewStore(":memory:")
	if err != nil {
		return nil, err
	}
	return &Server{
		listenAddr: listenAddr,
		store:      store,
	}, nil
}

func (s *Server) Run() error {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /events/retrieve", s.handleRetrieveEvents)

	mux.HandleFunc("POST /beginning", s.handleBeginning)
	mux.HandleFunc("POST /end", s.handleEnd)

	server := &http.Server{
		Addr:         s.listenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-shutdown
		log.Info().Msg("shutting down server")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	log.Info().Str("addr", s.listenAddr).Msg("starting server")
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

type beginningRequest struct {
	Name string `json:"name"`
}

type beginningResponse struct {
	ID string `json:"id"`
}

func (s *Server) handleBeginning(w http.ResponseWriter, r *http.Request) {
	var req beginningRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	id := uuid.New().String()
	if err := s.store.BeginEvent(id, req.Name); err != nil {
		log.Error().Err(err).Msg("failed to create event")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create event"})
		return
	}

	log.Info().Msgf("Event '%s' began, was given ID '%s'.", req.Name, id)
	writeJSON(w, http.StatusCreated, beginningResponse{ID: id})
}

type endRequest struct {
	ID string `json:"id"`
}

func (s *Server) handleEnd(w http.ResponseWriter, r *http.Request) {
	var req endRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.ID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id is required"})
		return
	}

	if err := s.store.EndEvent(req.ID); err != nil {
		log.Error().Err(err).Str("id", req.ID).Msg("failed to end event")
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	log.Info().Msgf("Event '%s' was ended.", req.ID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type retrieveResponse struct {
	Events []model.Event `json:"events"`
}

func (s *Server) handleRetrieveEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.store.RetrieveEvents()
	if err != nil {
		log.Error().Err(err).Msg("failed to retrieve events")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to retrieve events"})
		return
	}

	if events == nil {
		events = []model.Event{}
	}

	log.Info().Msgf("Retrieval of %d events.", len(events))
	writeJSON(w, http.StatusOK, retrieveResponse{Events: events})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

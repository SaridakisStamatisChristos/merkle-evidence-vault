package main

import (
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/SaridakisStamatisChristos/vault-api/handler"
	"github.com/SaridakisStamatisChristos/vault-api/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.Metrics)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ready"}`))
	})
	r.Handle("/metrics", middleware.MetricsHandler())

	// API routes (minimal in-memory implementation for tests)
	h := handler.NewIngestHandler()
	handler.StartCommitter(1 * time.Second)
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/evidence", h.Ingest)
		r.Get("/evidence/{id}", h.GetEvidence)
		r.Get("/evidence/{id}/proof", h.GetProof)

		// audit and checkpoint endpoints (JWT middleware)
		r.With(middleware.JWT).Get("/audit", h.GetAudit)
		r.With(middleware.JWT).Get("/checkpoints", h.GetCheckpointsHistory)
		r.With(middleware.JWT).Get("/checkpoints/latest", h.GetCheckpointsLatest)
		r.With(middleware.JWT).Get("/checkpoints/latest/verify", h.VerifyLatestCheckpoint)
		r.With(middleware.JWT).Get("/checkpoints/{treeSize}/verify", h.VerifyCheckpointByTreeSize)
	})

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8443"
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	log.Info().Msgf("starting vault-api on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Error().Err(err).Msg("server exited")
	}
}

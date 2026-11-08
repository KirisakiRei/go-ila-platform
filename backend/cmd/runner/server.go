package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"ila/backend/internal/runner"
)

// internalServer menerima permintaan dari backend-api.
//
// Port-nya tidak pernah dibuka ke publik; hanya di jaringan container
// (plans/09 §5). Setiap permintaan wajib membawa secret bersama.
type internalServer struct {
	svc    *runner.Service
	secret string
	logger *slog.Logger
}

// newInternalServer menyiapkan mux endpoint internal.
func newInternalServer(svc *runner.Service, secret string, logger *slog.Logger) *http.Server {
	handler := &internalServer{svc: svc, secret: secret, logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.health)
	mux.HandleFunc("/ready", handler.health)
	mux.HandleFunc("/internal/approve", handler.authenticate(handler.approve))
	mux.HandleFunc("/internal/skip", handler.authenticate(handler.skip))
	mux.HandleFunc("/internal/account/revoke", handler.authenticate(handler.revokeAccount))
	mux.HandleFunc("/internal/account/reload", handler.authenticate(handler.reloadAccount))

	return &http.Server{
		Addr:              ":" + envOr("RUNNER_HTTP_PORT", "8081"),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func (s *internalServer) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// authenticate menolak permintaan tanpa secret yang cocok.
func (s *internalServer) authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.secret == "" || r.Header.Get("X-Internal-Secret") != s.secret {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "tidak berwenang"})
			return
		}
		next(w, r)
	}
}

type approvePayload struct {
	UserID    string `json:"user_id"`
	SignalID  string `json:"signal_id"`
	AccountID string `json:"account_id"`
}

func (s *internalServer) approve(w http.ResponseWriter, r *http.Request) {
	var payload approvePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "payload tidak sah"})
		return
	}

	if payload.UserID == "" || payload.SignalID == "" || payload.AccountID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "data tidak lengkap"})
		return
	}

	reason, err := s.svc.Approve(r.Context(), payload.UserID, payload.SignalID, payload.AccountID)
	if err != nil {
		s.logger.Error("approve gagal",
			"user_id", payload.UserID, "signal_id", payload.SignalID, "error", err.Error())

		status := http.StatusInternalServerError
		if errors.Is(err, context.Canceled) {
			status = http.StatusRequestTimeout
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"reject_reason": string(reason)})
}

type skipPayload struct {
	UserID   string `json:"user_id"`
	SignalID string `json:"signal_id"`
}

func (s *internalServer) skip(w http.ResponseWriter, r *http.Request) {
	var payload skipPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "payload tidak sah"})
		return
	}

	if err := s.svc.Skip(r.Context(), payload.UserID, payload.SignalID); err != nil {
		s.logger.Error("skip gagal", "user_id", payload.UserID, "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type revokePayload struct {
	AccountID string `json:"account_id"`
}

// revokeAccount memutus sesi broker setelah kredensialnya dihapus.
func (s *internalServer) revokeAccount(w http.ResponseWriter, r *http.Request) {
	var payload revokePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.AccountID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "payload tidak sah"})
		return
	}

	s.svc.DropClient(payload.AccountID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// reloadAccount memaksa sesi broker dibuka ulang setelah kredensial berganti.
func (s *internalServer) reloadAccount(w http.ResponseWriter, r *http.Request) {
	var payload revokePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.AccountID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "payload tidak sah"})
		return
	}

	s.svc.DropClient(payload.AccountID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

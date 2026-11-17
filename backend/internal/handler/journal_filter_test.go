package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"net/url"
	"testing"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type journalFilterReader struct{ filters []repository.JournalFilter }

func (r *journalFilterReader) ListPositions(context.Context, string, repository.JournalFilter, int) ([]model.Position, error) {
	return nil, nil
}
func (r *journalFilterReader) Stats(context.Context, string, repository.JournalFilter) (repository.JournalStats, error) {
	return repository.JournalStats{}, nil
}
func (r *journalFilterReader) ListSkipped(_ context.Context, _ string, f repository.JournalFilter, _, _ int) ([]model.UserSignal, error) {
	r.filters = append(r.filters, f)
	return nil, nil
}
func (r *journalFilterReader) CountSkipped(_ context.Context, _ string, f repository.JournalFilter) (int, error) {
	r.filters = append(r.filters, f)
	return 0, nil
}

func TestJournalEvalStatusHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, status := range []string{"", "PENDING", "RUNNING", "TP", "SL", "NO_ENTRY", "UNKNOWN", "tp", "ALL", " RUNNING ", "TP' OR 1=1"} {
		t.Run(status, func(t *testing.T) {
			repo := &journalFilterReader{}
			svc := service.NewTradingService(nil, nil, nil, nil).WithJournal(repo, nil).Build()
			router := gin.New()
			router.Use(middleware.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))
			router.GET("/journal", NewTradingHandler(svc).Journal)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest("GET", "/journal?eval_status="+url.QueryEscape(status), nil))
			valid := status == "" || status == "PENDING" || status == "RUNNING" || status == "TP" || status == "SL" || status == "NO_ENTRY" || status == "UNKNOWN"
			if !valid {
				if rec.Code != 400 || len(repo.filters) != 0 {
					t.Fatalf("invalid status reached repository: HTTP %d %s", rec.Code, rec.Body.String())
				}
				var body struct {
					Error string `json:"error"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Error != "BAD_REQUEST" {
					t.Fatalf("error contract: %s", rec.Body.String())
				}
				return
			}
			if rec.Code != 200 || len(repo.filters) != 2 || repo.filters[0].EvalStatus != status || repo.filters[1].EvalStatus != status {
				t.Fatalf("filter not propagated: HTTP %d filters=%+v", rec.Code, repo.filters)
			}
			var body struct {
				Data dto.JournalResponse `json:"data"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Data.Filters.EvalStatus != status {
				t.Fatalf("filter echo missing: %s", rec.Body.String())
			}
		})
	}
}

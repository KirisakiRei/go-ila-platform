package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ila/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func TestManualJournalReviewHandlerRegistrationAndBoundaries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			c.Status(http.StatusBadRequest)
		}
	})
	// Missing authentication must stop before touching any dependency.
	NewManualJournalReviewHandler(service.NewManualJournalReviewService(nil, nil, nil, false, "")).Register(r.Group("/api"))
	for _, test := range []struct{ method, path, body string }{
		{"GET", "/api/manual-journal/review?account_id=bad", ""},
		{"GET", "/api/manual-journal/review/settings", ""},
		{"PUT", "/api/manual-journal/review/settings", `{"ai_enabled":false}`},
		{"PUT", "/api/manual-journal/review/settings", `{}`},
		{"PUT", "/api/manual-journal/review/settings", strings.Repeat("x", 1025)},
	} {
		t.Run(test.method+test.path+test.body[:min(len(test.body), 10)], func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("boundary returned %d", rec.Code)
			}
		})
	}
}

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestManualJournalHTTPContracts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.AutoMigrate(&model.ManualJournalAccount{}, &model.ManualJournalCapital{}, &model.ManualJournalEntry{}, &model.ManualJournalScreenshot{}, &model.ManualJournalImportToken{}, &model.ManualJournalReview{}, &model.ManualJournalReviewSetting{}); err != nil {
		t.Fatal(err)
	}
	svc := service.NewManualJournalService(repository.NewManualJournalRepository(db), t.TempDir())
	user := uuid.NewString()
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextUserID, c.GetHeader("X-Test-User"))
		c.Next()
		if len(c.Errors) > 0 {
			_ = httpx.Fail(c, c.Errors.Last().Err)
		}
	})
	NewManualJournalHandler(svc).Register(r.Group("/api/v1"))
	request := func(method, path, user string, body any) *httptest.ResponseRecorder {
		t.Helper()
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, "/api/v1/manual-journal"+path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User", user)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	w := request("POST", "/accounts", user, dto.ManualJournalAccountInput{Name: "Test", Currency: "USD", InitialCapital: "100", InitialDate: "2026-01-01"})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	var out struct {
		Success bool                             `json:"success"`
		Data    dto.ManualJournalAccountResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || !out.Success || out.Data.ID == "" {
		t.Fatalf("envelope: %s %v", w.Body, err)
	}
	if w := request("GET", "/accounts/"+out.Data.ID, uuid.NewString(), nil); w.Code != http.StatusNotFound {
		t.Fatalf("owner: %d %s", w.Code, w.Body)
	}
	if w := request("GET", "/accounts", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("unauth: %d %s", w.Code, w.Body)
	}
	if w := request("GET", "/accounts?per_page=101", user, nil); w.Code != http.StatusBadRequest {
		t.Fatalf("limit: %d %s", w.Code, w.Body)
	}
	if w := request("GET", "/entries?from=2026-99-99", user, nil); w.Code != http.StatusBadRequest {
		t.Fatalf("date: %d %s", w.Code, w.Body)
	}
	if w := request("POST", "/accounts", user, map[string]any{"name": "x", "currency": "USD", "initial_capital": 12, "initial_date": "2026-01-01"}); w.Code != http.StatusBadRequest {
		t.Fatalf("numeric money: %d %s", w.Code, w.Body)
	}
	if w := request("GET", "/accounts?page=1&per_page=1", user, nil); w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`"total_items":1`)) {
		t.Fatalf("pagination: %d %s", w.Code, w.Body)
	}
	if w := request("GET", "/export", user, nil); w.Code != http.StatusOK || w.Header().Get("Content-Type") != "text/csv; charset=utf-8" {
		t.Fatalf("export: %d %s", w.Code, w.Body)
	}
	draft, err := svc.SaveEntry(context.Background(), user, "", dto.ManualJournalEntryInput{AccountID: out.Data.ID, Status: "draft"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := svc.ExportCSV(context.Background(), user, dto.ManualJournalFilter{})
	if err != nil {
		t.Fatal(err)
	}
	previewResponse := request("POST", "/import/preview", user, dto.ManualJournalCSVInput{CSV: string(data)})
	if previewResponse.Code != http.StatusOK {
		t.Fatalf("preview: %d %s", previewResponse.Code, previewResponse.Body)
	}
	var preview struct {
		Data dto.ManualJournalPreview `json:"data"`
	}
	if err := json.Unmarshal(previewResponse.Body.Bytes(), &preview); err != nil || preview.Data.Token == "" || len(preview.Data.Rows) != 1 {
		t.Fatalf("preview contract: %s %v", previewResponse.Body, err)
	}
	if w := request("POST", "/import/confirm", user, dto.ManualJournalCSVInput{CSV: string(data)}); w.Code != http.StatusBadRequest {
		t.Fatalf("token required: %d %s", w.Code, w.Body)
	}
	confirm := dto.ManualJournalConfirmInput{CSV: string(data), Token: preview.Data.Token}
	if w := request("POST", "/import/confirm", user, confirm); w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`"count":1`)) {
		t.Fatalf("confirm contract: %d %s", w.Code, w.Body)
	}
	if w := request("POST", "/import/confirm", user, confirm); w.Code != http.StatusConflict {
		t.Fatalf("replay contract: %d %s", w.Code, w.Body)
	}
	// The API rejects naive local time rather than silently assuming UTC.
	if w := request("PUT", "/entries/"+draft.ID, user, map[string]any{"account_id": out.Data.ID, "status": "open", "opened_at": "2026-01-01T09:00:00"}); w.Code != http.StatusBadRequest {
		t.Fatalf("naive time: %d %s", w.Code, w.Body)
	}
	// The API rejects inverted Buy/Sell SL/TP against Entry.
	if w := request("POST", "/entries", user, map[string]any{"account_id": out.Data.ID, "status": "draft", "side": "long", "entry_price": "3500", "take_profit": "3480"}); w.Code != http.StatusBadRequest {
		t.Fatalf("inverted long tp: %d %s", w.Code, w.Body)
	}
	if w := request("POST", "/entries", user, map[string]any{"account_id": out.Data.ID, "status": "draft", "side": "short", "entry_price": "3500", "stop_loss": "3490"}); w.Code != http.StatusBadRequest {
		t.Fatalf("inverted short sl: %d %s", w.Code, w.Body)
	}
	if w := request("POST", "/entries", user, map[string]any{"account_id": out.Data.ID, "status": "draft", "side": "long", "entry_price": "3500", "stop_loss": "3490", "take_profit": "3520"}); w.Code != http.StatusCreated {
		t.Fatalf("valid long levels: %d %s", w.Code, w.Body)
	}
	// Default DELETE archives; permanent=true deletes the journal and its child entries.
	if w := request("DELETE", "/accounts/"+out.Data.ID, user, nil); w.Code != http.StatusOK {
		t.Fatalf("archive: %d %s", w.Code, w.Body)
	}
	if w := request("GET", "/accounts/"+out.Data.ID, user, nil); w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`"archived":true`)) {
		t.Fatalf("archived check: %d %s", w.Code, w.Body)
	}
	if w := request("DELETE", "/accounts/"+out.Data.ID+"?permanent=true", uuid.NewString(), nil); w.Code != http.StatusNotFound {
		t.Fatalf("permanent owner check: %d %s", w.Code, w.Body)
	}
	if w := request("DELETE", "/accounts/"+out.Data.ID+"?permanent=true", user, nil); w.Code != http.StatusOK {
		t.Fatalf("permanent delete: %d %s", w.Code, w.Body)
	}
	if w := request("GET", "/accounts/"+out.Data.ID, user, nil); w.Code != http.StatusNotFound {
		t.Fatalf("account still exists after delete: %d %s", w.Code, w.Body)
	}
	if w := request("GET", "/entries/"+draft.ID, user, nil); w.Code != http.StatusNotFound {
		t.Fatalf("entry still exists after account delete: %d %s", w.Code, w.Body)
	}
}

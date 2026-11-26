package router

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"ila/backend/internal/service"
	"ila/backend/pkg/apperror"

	"gorm.io/gorm"
)

func TestManualJournalRoutesRequireSession(t *testing.T) {
	r := NewBackendAPI(testConfig(t), testDB(t), testLogger())
	want := map[string]bool{}
	for _, resource := range []string{"accounts", "capital", "entries"} {
		base := "/api/v1/manual-journal/" + resource
		for _, route := range []string{"GET " + base, "POST " + base, "GET " + base + "/:id", "PUT " + base + "/:id", "DELETE " + base + "/:id"} {
			want[route] = false
		}
	}
	for _, route := range []string{
		"GET /stats", "GET /insights", "GET /entries/:id/screenshots", "POST /entries/:id/screenshots",
		"GET /entries/:id/screenshots/:sid", "DELETE /entries/:id/screenshots/:sid",
		"POST /import/preview", "POST /import/confirm", "GET /export",
		"GET /review", "GET /review/settings", "PUT /review/settings",
	} {
		method, path, _ := strings.Cut(route, " ")
		want[method+" /api/v1/manual-journal"+path] = false
	}
	for _, route := range r.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; !ok {
			continue
		}
		want[key] = true
		t.Run(key, func(t *testing.T) {
			path := strings.ReplaceAll(strings.ReplaceAll(route.Path, ":id", "entry"), ":sid", "image")
			if w := performRequest(r, route.Method, path); w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401: %s", w.Code, w.Body.String())
			}
		})
	}
	for route, found := range want {
		if !found {
			t.Errorf("missing route: %s", route)
		}
	}
	for _, path := range []string{"/manual-journal/image.png", "/uploads/manual-journal/image.png"} {
		if w := performRequest(r, http.MethodGet, path); w.Code != http.StatusNotFound {
			t.Errorf("private upload path %s: status = %d, want 404", path, w.Code)
		}
	}
}

func TestManualJournalWorkerImmediateBoundedPollAndShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		runManualJournalReviews(ctx, func(pollCtx context.Context) error {
			deadline, ok := pollCtx.Deadline()
			if !ok || time.Until(deadline) > time.Minute {
				t.Error("poll must have a deadline of at most one minute")
			}
			close(started)
			<-pollCtx.Done()
			return pollCtx.Err()
		}, testLogger())
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not poll immediately")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop on parent cancellation")
	}
}

func TestManualJournalCleanupWorkerRegisteredDespiteReviewFailure(t *testing.T) {
	db := testDB(t)
	cleanup := make(chan struct{}, 1)
	review := make(chan struct{}, 1)
	if err := db.Callback().Query().Before("gorm:query").Register("test:manual-journal-workers", func(tx *gorm.DB) {
		switch tx.Statement.Table {
		case "manual_journal_accounts":
			tx.AddError(errors.New("review unavailable"))
			select {
			case review <- struct{}{}:
			default:
			}
		case "manual_journal_screenshots":
			deadline, ok := tx.Statement.Context.Deadline()
			if !ok || time.Until(deadline) > time.Minute {
				t.Error("cleanup query must have a deadline of at most one minute")
			}
			select {
			case cleanup <- struct{}{}:
			default:
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	var workers []func(context.Context)
	NewBackendAPI(testConfig(t), db, testLogger(), DenganPekerjaanLatar(func(v []func(context.Context)) {
		workers = v
	}))
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for _, worker := range workers {
		wg.Add(1)
		go func(worker func(context.Context)) {
			defer wg.Done()
			worker(ctx)
		}(worker)
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("registered workers did not stop on parent cancellation")
		}
	}()
	select {
	case <-review:
	case <-time.After(time.Second):
		t.Fatal("registered review worker did not encounter the injected failure")
	}
	select {
	case <-cleanup:
	case <-time.After(time.Second):
		t.Fatal("registered cleanup worker did not query screenshot tombstones despite review failure")
	}
}

func TestManualJournalReviewAdapterPreservesValidation(t *testing.T) {
	journal := manualJournalReviewJournal{service.NewManualJournalService(nil, "")}
	for _, tc := range []struct {
		user, account string
		want          error
	}{
		{"", "account", apperror.ErrUnauthorized},
		{"user", "invalid-id", apperror.ErrBadRequest},
	} {
		if v, err := journal.GetAccount(context.Background(), tc.user, tc.account); v != nil || !errors.Is(err, tc.want) {
			t.Errorf("GetAccount(%q, %q) = %v, %v; want nil, %v", tc.user, tc.account, v, err, tc.want)
		}
	}
}

func TestManualJournalWorkerAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runManualJournalReviews(ctx, func(context.Context) error {
		t.Fatal("canceled worker must not poll")
		return nil
	}, testLogger())
}

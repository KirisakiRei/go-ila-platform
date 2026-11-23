package repository

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ila/backend/internal/model"
)

func openAssistantTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("membuka sqlite uji: %v", err)
	}
	if err := db.AutoMigrate(&model.AISession{}, &model.AIMessage{}, &model.AIUserMemory{}); err != nil {
		t.Fatalf("migrasi tabel asisten: %v", err)
	}
	return db
}

func TestAssistantRepository_SingleSessionAndCompaction(t *testing.T) {
	ctx := context.Background()
	db := openAssistantTestDB(t)
	repo := NewAssistantRepository(db)

	s1, err := repo.GetOrCreateSession(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetOrCreateSession: %v", err)
	}
	s2, err := repo.GetOrCreateSession(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetOrCreateSession kedua: %v", err)
	}
	if s1.ID != s2.ID {
		t.Fatalf("diharapkan 1 sesi per user, dapat %s dan %s", s1.ID, s2.ID)
	}

	t0 := time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)
	saved, err := repo.AppendMessages(ctx, s1.ID, "user-1", t0, []model.AIMessage{
		{Role: "user", Content: "Halo"},
		{Role: "assistant", Content: "", ToolCallsJSON: `[{"id":"call_1"}]`},
		{Role: "tool", Content: `{"ok":true}`, ToolCallID: "call_1", ToolName: "get_signals"},
		{Role: "assistant", Content: "Ada 1 sinyal XAUUSD."},
	})
	if err != nil {
		t.Fatalf("AppendMessages: %v", err)
	}
	if len(saved) != 4 || saved[0].Seq != 1 || saved[3].Seq != 4 {
		t.Fatalf("urutan seq salah: %+v", saved)
	}

	display, hasMore, err := repo.ListDisplayMessages(ctx, s1.ID, 0, 10)
	if err != nil {
		t.Fatalf("ListDisplayMessages: %v", err)
	}
	if hasMore || len(display) != 2 {
		t.Fatalf("pesan tampilan diharapkan 2 (tanpa tool/empty), dapat %d (hasMore=%v)", len(display), hasMore)
	}
	if display[0].Content != "Halo" || display[1].Content != "Ada 1 sinyal XAUUSD." {
		t.Fatalf("isi tampilan tak sesuai: %+v", display)
	}

	if err := repo.ApplyCompaction(ctx, s1.ID, "user-1", 3, "Ringkasan awal", "- Pantau XAUUSD", t0.Add(time.Minute)); err != nil {
		t.Fatalf("ApplyCompaction: %v", err)
	}

	active, err := repo.ListActiveMessages(ctx, s1.ID, 20)
	if err != nil {
		t.Fatalf("ListActiveMessages: %v", err)
	}
	if len(active) != 1 || active[0].Seq != 4 {
		t.Fatalf("pesan aktif setelah compact diharapkan seq 4 saja, dapat %+v", active)
	}

	mem, err := repo.GetUserMemory(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetUserMemory: %v", err)
	}
	if mem.MemoryMarkdown != "- Pantau XAUUSD" {
		t.Fatalf("memori tak tersimpan: %q", mem.MemoryMarkdown)
	}
}

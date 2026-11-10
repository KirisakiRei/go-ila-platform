package dto_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"ila/backend/internal/dto"
)

// Sinyal harus menyertakan updated_at terpisah dari published_at. UI memakai
// updated_at untuk label "Update" agar tidak tampak macet pada waktu terbit
// pertama saat engine hanya menyegarkan ringkasan.
func TestSignalResponseMenyertakanUpdatedAt(t *testing.T) {
	terbit := time.Date(2026, 11, 9, 16, 5, 0, 0, time.UTC)
	diperbarui := time.Date(2026, 11, 10, 14, 0, 0, 0, time.UTC)

	resp := dto.SignalResponse{
		SignalID:    "sig-1",
		PublishedAt: terbit,
		UpdatedAt:   diperbarui,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var hasil map[string]any
	if err := json.Unmarshal(data, &hasil); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	updated, ada := hasil["updated_at"]
	if !ada {
		t.Fatalf("updated_at tidak ada pada payload: %s", data)
	}
	if updated != diperbarui.Format(time.RFC3339) {
		t.Errorf("updated_at = %v, ingin %s", updated, diperbarui.Format(time.RFC3339))
	}
	if !strings.Contains(string(data), `"published_at"`) {
		t.Errorf("published_at harus tetap ada: %s", data)
	}
}

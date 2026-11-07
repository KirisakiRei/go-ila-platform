package runner

import (
	"reflect"
	"testing"
)

func TestDiffRequirements(t *testing.T) {
	cases := []struct {
		name         string
		old, current []string
		added        []string
		removed      []string
	}{
		{
			name:    "langganan pertama",
			old:     nil,
			current: []string{"EURUSD", "XAUUSD"},
			added:   []string{"EURUSD", "XAUUSD"},
		},
		{
			name:    "tidak ada perubahan",
			old:     []string{"XAUUSD"},
			current: []string{"XAUUSD"},
		},
		{
			name:    "simbol ditambah",
			old:     []string{"XAUUSD"},
			current: []string{"XAUUSD", "USTEC"},
			added:   []string{"USTEC"},
		},
		{
			name:    "simbol dilepas",
			old:     []string{"EURUSD", "XAUUSD"},
			current: []string{"XAUUSD"},
			removed: []string{"EURUSD"},
		},
		{
			name:    "semua berubah",
			old:     []string{"EURUSD"},
			current: []string{"USTEC"},
			added:   []string{"USTEC"},
			removed: []string{"EURUSD"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			added, removed := DiffRequirements(tc.old, tc.current)

			if !reflect.DeepEqual(added, tc.added) {
				t.Fatalf("added = %v, ingin %v", added, tc.added)
			}
			if !reflect.DeepEqual(removed, tc.removed) {
				t.Fatalf("removed = %v, ingin %v", removed, tc.removed)
			}
		})
	}
}

// Beberapa user pada simbol sama tetap menghasilkan satu entri.
func TestDiffRequirementsDeduplicates(t *testing.T) {
	added, _ := DiffRequirements(nil, []string{"XAUUSD", "XAUUSD"})
	if len(added) != 1 {
		t.Fatalf("added = %v, ingin satu entri", added)
	}
}

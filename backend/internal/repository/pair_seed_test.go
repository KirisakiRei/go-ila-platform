package repository

import (
	"sort"
	"strings"
	"testing"
)

// Seed korelasi harus menunjuk simbol yang benar-benar ada di katalog: admin
// service menolak simbol asing, dan data seed tidak melewati validasi itu.
func TestDefaultPairSeedsKorelasiKonsisten(t *testing.T) {
	_, pairs := DefaultPairSeeds()

	known := make(map[string]bool, len(pairs))
	for _, p := range pairs {
		known[p.Symbol] = true
	}

	for _, p := range pairs {
		if p.CorrelatedSymbols == "" {
			continue
		}

		seen := make(map[string]bool)
		var list []string
		for _, part := range strings.Split(p.CorrelatedSymbols, ",") {
			value := strings.TrimSpace(part)
			if value == "" {
				t.Fatalf("%s: entri korelasi kosong pada %q", p.Symbol, p.CorrelatedSymbols)
			}
			if value != strings.ToUpper(value) {
				t.Errorf("%s: korelasi %q harus huruf besar", p.Symbol, value)
			}
			if seen[value] {
				t.Errorf("%s: korelasi %q berulang", p.Symbol, value)
			}
			seen[value] = true
			if value == p.Symbol {
				t.Errorf("%s: korelasi diri sendiri tidak boleh ada", p.Symbol)
			}
			if !known[value] {
				t.Errorf("%s: korelasi %q tidak ada di katalog seed", p.Symbol, value)
			}
			list = append(list, value)
		}

		if !sort.StringsAreSorted(list) {
			t.Errorf("%s: korelasi %q harus urut abjad", p.Symbol, p.CorrelatedSymbols)
		}
	}
}

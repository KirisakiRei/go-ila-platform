package runner

import "testing"

func TestEvalCandleSeconds(t *testing.T) {
	for tf, want := range map[string]int64{"M15": 900, "15M": 900, "H4": 14400, "4H": 14400, "D": 86400, "D1": 86400, "1D": 86400, "W": 604800, "W1": 604800, "1W": 604800, " m15 ": 900, "": 0, "M0": 0, "-1H": 0, "H1x": 0, "H+1": 0, "15MM": 0} {
		if got := evalCandleSeconds(tf); got != want {
			t.Errorf("%q: got %d want %d", tf, got, want)
		}
	}
}

package middleware

import "testing"

func TestSafePathMenyamarkanTokenApprove(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "token saja",
			input: "/api/v1/approve/RAHASIA123",
			want:  "/api/v1/approve/<disembunyikan>",
		},
		{
			name:  "dengan sub-path confirm",
			input: "/api/v1/approve/RAHASIA123/confirm",
			want:  "/api/v1/approve/<disembunyikan>/confirm",
		},
		{
			name:  "dengan sub-path decline",
			input: "/api/v1/approve/RAHASIA123/decline",
			want:  "/api/v1/approve/<disembunyikan>/decline",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := safePath(tc.input); got != tc.want {
				t.Fatalf("safePath(%q) = %q, ingin %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestSafePathTidakMengubahPathBiasa(t *testing.T) {
	for _, path := range []string{
		"/api/v1/me",
		"/api/v1/accounts",
		"/api/v1/subscriptions/abc",
		"/health",
		"/approve/",
	} {
		if got := safePath(path); got != path {
			t.Fatalf("safePath(%q) berubah menjadi %q", path, got)
		}
	}
}

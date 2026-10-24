package gitlab

import "testing"

func TestNormalizeState(t *testing.T) {
	tests := map[string]string{
		"opened": "open",
		"closed": "closed",
		"merged": "merged",
		"locked": "locked",
	}
	for in, want := range tests {
		if got := normalizeState(in); got != want {
			t.Errorf("normalizeState(%q) = %q, want %q", in, got, want)
		}
	}
}

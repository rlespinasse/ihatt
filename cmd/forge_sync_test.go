package cmd

import "testing"

func TestHostKind(t *testing.T) {
	kinds := map[string]string{"github.com": "github", "git.acme.io": "gitlab", "code.acme.io": "github"}
	tests := []struct {
		host     string
		wantKind string
		wantOK   bool
	}{
		{"github.com", "github", true},
		{"github.acme.io", "github", true},
		{"git.gitlab.acme.io", "gitlab", true},
		{"git.acme.io", "gitlab", true},
		{"code.acme.io", "github", true},
		{"git.example.test", "", false},
		{"gitlab-github.acme.io", "", false},
	}
	for _, tt := range tests {
		kind, ok := hostKind(kinds, tt.host)
		if kind != tt.wantKind || ok != tt.wantOK {
			t.Errorf("hostKind(%q) = (%q, %v), want (%q, %v)", tt.host, kind, ok, tt.wantKind, tt.wantOK)
		}
	}
}

package cmd

import (
	"testing"

	"github.com/rlespinasse/ihatt/internal/config"
)

func TestLookupHost(t *testing.T) {
	hosts := forgeHosts{
		"github.com":          {Kind: "github"},
		"git.acme.io":         {Kind: "gitlab"},
		"code.acme.io":        {Kind: "github"},
		"ssh.sources.acme.io": {Kind: "gitlab", APIHost: "sources.acme.io"},
	}
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
		fh, ok := lookupHost(hosts, tt.host)
		if fh.Kind != tt.wantKind || ok != tt.wantOK {
			t.Errorf("lookupHost(%q) = (%q, %v), want (%q, %v)", tt.host, fh.Kind, ok, tt.wantKind, tt.wantOK)
		}
	}
}

func TestForgeHostsFromKeepsAPIHost(t *testing.T) {
	cfg := &config.Config{Forges: []config.Forge{{Host: "SSH.Sources.Acme.io", Kind: "gitlab", APIHost: "Sources.Acme.io"}}}
	hosts, err := forgeHostsFrom(cfg)
	if err != nil {
		t.Fatal(err)
	}
	fh, ok := lookupHost(hosts, "ssh.sources.acme.io")
	if !ok || fh.Kind != "gitlab" || fh.APIHost != "sources.acme.io" {
		t.Errorf("got (%+v, %v), want gitlab with API host sources.acme.io", fh, ok)
	}
}

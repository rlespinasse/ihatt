package config

import (
	"reflect"
	"testing"
)

func TestSaveKeepsForgesAcrossLoad(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	want := &Config{
		ScanRoots: []string{"/code"},
		Forges: []Forge{
			{Host: "github.acme.io", Kind: "github"},
			{Host: "git.acme.io", Kind: "gitlab"},
		},
	}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.ScanRoots, want.ScanRoots) {
		t.Errorf("ScanRoots = %v, want %v", got.ScanRoots, want.ScanRoots)
	}
	if !reflect.DeepEqual(got.Forges, want.Forges) {
		t.Errorf("Forges = %+v, want %+v", got.Forges, want.Forges)
	}
}

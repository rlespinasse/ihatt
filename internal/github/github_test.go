package github

import (
	"slices"
	"testing"

	"github.com/rlespinasse/ihatt/internal/forge"
)

func TestListArgsTargetsHost(t *testing.T) {
	r := forge.Remote{Host: "github.acme.io", Owner: "team", Repo: "repo"}
	args := listArgs("pr", r, forge.Plan{}, "number")
	if i := slices.Index(args, "--repo"); i < 0 || args[i+1] != "github.acme.io/team/repo" {
		t.Errorf("args = %v, want --repo github.acme.io/team/repo", args)
	}
}

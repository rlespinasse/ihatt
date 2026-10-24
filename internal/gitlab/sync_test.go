package gitlab

import (
	"testing"

	"github.com/rlespinasse/ihatt/internal/forge"
)

func TestDecodePagesJoinsConcatenatedArrays(t *testing.T) {
	data := []byte(`[{"iid":1},{"iid":2}]
[{"iid":3}]
[]`)
	got, err := decodePages[glIssue]([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].IID != 1 || got[2].IID != 3 {
		t.Fatalf("got %+v, want iids 1,2,3", got)
	}
}

func TestMergeSHAPrefersSquash(t *testing.T) {
	squash, merge := "sq", "mg"
	tests := []struct {
		name string
		mr   glMR
		want string
	}{
		{"squashed", glMR{SquashSHA: &squash, MergeCommitSHA: &merge}, "sq"},
		{"merge commit", glMR{MergeCommitSHA: &merge}, "mg"},
		{"empty squash falls back", glMR{SquashSHA: new(string), MergeCommitSHA: &merge}, "mg"},
		{"open MR", glMR{}, ""},
	}
	for _, tt := range tests {
		if got := mergeSHA(tt.mr); got != tt.want {
			t.Errorf("%s: mergeSHA = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestProjectPathEncodesAndIncrements(t *testing.T) {
	full := projectPath(forge.Remote{Owner: "group/sub", Repo: "proj"}, "issues", forge.Plan{})
	if want := "projects/group%2Fsub%2Fproj/issues?"; full[:len(want)] != want {
		t.Errorf("path = %q, want prefix %q", full, want)
	}
	if containsUpdatedAfter(full) {
		t.Error("full sync must not send updated_after")
	}
}

func containsUpdatedAfter(p string) bool {
	for i := 0; i+len("updated_after=") <= len(p); i++ {
		if p[i:i+len("updated_after=")] == "updated_after=" {
			return true
		}
	}
	return false
}

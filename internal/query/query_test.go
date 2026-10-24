package query

import (
	"testing"

	"github.com/rlespinasse/ihatt/internal/model"
)

func TestItemFilter(t *testing.T) {
	items := []*model.GitHubItem{
		{Number: 1, Type: "issue", State: "open", Labels: []string{"bug"}},
		{Number: 2, Type: "pr", State: "merged", Provider: "gitlab", Labels: []string{"Bug"}},
		{Number: 3, Type: "pr", State: "open", Provider: "gitlab"},
	}
	tests := []struct {
		name   string
		filter ItemFilter
		want   []int
	}{
		{"empty matches all", ItemFilter{}, []int{1, 2, 3}},
		{"provider defaults to github", ItemFilter{Provider: "github"}, []int{1}},
		{"gitlab", ItemFilter{Provider: "gitlab"}, []int{2, 3}},
		{"state case-insensitive", ItemFilter{State: "OPEN"}, []int{1, 3}},
		{"type and provider", ItemFilter{Type: "pr", Provider: "gitlab"}, []int{2, 3}},
		{"label case-insensitive", ItemFilter{Label: "bug"}, []int{1, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []int
			for _, item := range tt.filter.Apply(items) {
				got = append(got, item.Number)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestTypeLabelForGitLabMR(t *testing.T) {
	if l := (&model.GitHubItem{Type: "pr", Provider: "gitlab"}).TypeLabel(); l != "MR" {
		t.Errorf("gitlab pr label = %q, want MR", l)
	}
	if l := (&model.GitHubItem{Type: "pr"}).TypeLabel(); l != "PR" {
		t.Errorf("github pr label = %q, want PR", l)
	}
}

package gitlab

import "testing"

func TestParseGitLabRemote(t *testing.T) {
	tests := []struct {
		name      string
		remote    string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{"scp ssh", "git@gitlab.com:group/project.git", "group", "project", false},
		{"scp ssh without .git", "git@gitlab.com:group/project", "group", "project", false},
		{"ssh url", "ssh://git@gitlab.com/group/project.git", "group", "project", false},
		{"ssh url with port", "ssh://git@gitlab.com:2222/group/project.git", "group", "project", false},
		{"https", "https://gitlab.com/group/project.git", "group", "project", false},
		{"https without .git", "https://gitlab.com/group/project", "group", "project", false},
		{"nested groups", "https://gitlab.com/group/sub/project.git", "group/sub", "project", false},
		{"nested groups scp", "git@gitlab.com:group/sub/deeper/project.git", "group/sub/deeper", "project", false},
		{"github https rejected", "https://github.com/owner/repo.git", "", "", true},
		{"github scp rejected", "git@github.com:owner/repo.git", "", "", true},
		{"no project", "https://gitlab.com/group", "", "", true},
		{"no path", "https://gitlab.com/", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseGitLabRemote(tt.remote)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseGitLabRemote(%q) error = %v, wantErr %v", tt.remote, err, tt.wantErr)
			}
			if owner != tt.wantOwner || repo != tt.wantRepo {
				t.Errorf("ParseGitLabRemote(%q) = (%q, %q), want (%q, %q)",
					tt.remote, owner, repo, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}

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

package forge

import "testing"

func TestParseRemote(t *testing.T) {
	tests := []struct {
		name   string
		remote string
		want   Remote
		wantOK bool
	}{
		{"github scp ssh", "git@github.com:owner/repo.git", Remote{"github.com", "owner", "repo"}, true},
		{"github https", "https://github.com/owner/repo.git", Remote{"github.com", "owner", "repo"}, true},
		{"github enterprise scp", "git@github.acme.io:team/repo.git", Remote{"github.acme.io", "team", "repo"}, true},
		{"github enterprise https", "https://github.acme.io/team/repo", Remote{"github.acme.io", "team", "repo"}, true},
		{"gitlab scp without .git", "git@gitlab.com:group/project", Remote{"gitlab.com", "group", "project"}, true},
		{"gitlab ssh url with port", "ssh://git@gitlab.com:2222/group/project.git", Remote{"gitlab.com", "group", "project"}, true},
		{"self-hosted gitlab https", "https://GitLab.Acme.io/group/project.git", Remote{"gitlab.acme.io", "group", "project"}, true},
		{"nested groups", "https://gitlab.com/group/sub/deeper/project.git", Remote{"gitlab.com", "group/sub/deeper", "project"}, true},
		{"nested groups scp", "git@gitlab.acme.io:group/sub/project.git", Remote{"gitlab.acme.io", "group/sub", "project"}, true},
		{"no project", "https://gitlab.com/group", Remote{}, false},
		{"no path", "https://gitlab.com/", Remote{}, false},
		{"local path", "/srv/git/repo.git", Remote{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRemote(tt.remote)
			if (err == nil) != tt.wantOK {
				t.Fatalf("ParseRemote(%q) error = %v, wantOK %v", tt.remote, err, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("ParseRemote(%q) = %+v, want %+v", tt.remote, got, tt.want)
			}
		})
	}
}

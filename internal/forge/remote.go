package forge

import (
	"fmt"
	"net/url"
	"strings"
)

// Remote is the forge project behind a git remote URL.
type Remote struct {
	Host  string // lowercase hostname, without any port
	Owner string // owner or GitLab namespace; nested groups make it contain "/"
	Repo  string
}

// ParseRemote reads the host and project path from an https, ssh:// or
// scp-style (git@host:path) remote URL.
func ParseRemote(remoteURL string) (Remote, error) {
	var host, path string
	if strings.Contains(remoteURL, "://") {
		u, err := url.Parse(remoteURL)
		if err != nil {
			return Remote{}, err
		}
		host, path = u.Hostname(), u.Path
	} else if i := strings.Index(remoteURL, ":"); i > 0 && !strings.Contains(remoteURL[:i], "/") {
		host, path = remoteURL[:i], remoteURL[i+1:]
		if at := strings.LastIndex(host, "@"); at >= 0 {
			host = host[at+1:]
		}
	} else {
		return Remote{}, fmt.Errorf("not a git remote URL: %s", remoteURL)
	}

	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	i := strings.LastIndex(path, "/")
	if host == "" || i <= 0 || i == len(path)-1 {
		return Remote{}, fmt.Errorf("invalid project path in remote URL: %s", remoteURL)
	}
	return Remote{Host: strings.ToLower(host), Owner: path[:i], Repo: path[i+1:]}, nil
}

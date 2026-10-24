package cmd

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/rlespinasse/ihatt/internal/model"
	"golang.org/x/term"

	"github.com/rlespinasse/ihatt/internal/config"
	"github.com/rlespinasse/ihatt/internal/forge"
	gitscanner "github.com/rlespinasse/ihatt/internal/git"
	ghclient "github.com/rlespinasse/ihatt/internal/github"
	"github.com/rlespinasse/ihatt/internal/gitlab"
	"github.com/rlespinasse/ihatt/internal/store"
	"github.com/spf13/cobra"
)

// forgeSyncFunc syncs issues, PRs/MRs and CI runs of one repo and returns their counts.
type forgeSyncFunc func(s store.Store, repoID string, r forge.Remote, opts forge.Options) (int, int, int, error)

// forgeKinds maps each configured forge host to its kind, "github" or "gitlab".
// The config's forges list adds hosts or overrides the built-in ones.
func forgeKinds(cfg *config.Config) (map[string]string, error) {
	kinds := map[string]string{"github.com": "github", "gitlab.com": "gitlab"}
	for _, f := range cfg.Forges {
		if !validForgeKind(f.Kind) {
			return nil, fmt.Errorf("forge %s: kind must be \"github\", \"gitlab\" or \"unsupported\", got %q", f.Host, f.Kind)
		}
		kinds[strings.ToLower(f.Host)] = f.Kind
	}
	return kinds, nil
}

// hostKind returns the kind of a forge host: the configured kind when the host
// is listed, otherwise the kind named in the hostname. ok is false when neither
// applies, or when the hostname names both forges.
func hostKind(kinds map[string]string, host string) (kind string, ok bool) {
	if kind, ok := kinds[host]; ok {
		return kind, true
	}
	hasGitHub := strings.Contains(host, "github")
	hasGitLab := strings.Contains(host, "gitlab")
	switch {
	case hasGitHub && !hasGitLab:
		return "github", true
	case hasGitLab && !hasGitHub:
		return "gitlab", true
	}
	return "", false
}

// syncerFor returns the forge client that syncs hosts of one kind.
func syncerFor(kind string) forgeSyncFunc {
	if kind == "gitlab" {
		return gitlab.SyncRepo
	}
	return ghclient.SyncRepo
}

// runSync is the loop behind `sync`, `github sync` and `gitlab sync`. It acts on
// the repos in scope (see repoScope): it indexes their commits when commits is
// set, and syncs the forge data of repos whose remote host is one of kinds.
func runSync(cmd *cobra.Command, commits bool, kinds ...string) error {
	opts := forge.Options{}
	opts.Full, _ = cmd.Flags().GetBool("full")
	opts.Comments, _ = cmd.Flags().GetBool("with-comments")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	hostKinds, err := forgeKinds(cfg)
	if err != nil {
		return err
	}

	s, err := store.OpenBoltStore(config.DBPath())
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer s.Close()

	repos, err := s.ListRepos()
	if err != nil {
		return err
	}

	if len(repos) == 0 {
		return fmt.Errorf("no repositories tracked; use 'ihatt scan' or 'ihatt repo add' first")
	}

	repoFilter, scope := repoScope(cmd, repos)
	fmt.Printf("Scope: %s\n", scope)

	hostKinds, err = classifyUnknownHosts(cfg, hostKinds, repoFilter, repos)
	if err != nil {
		return err
	}

	for _, repo := range repos {
		if repoFilter != "" && repo.Name != repoFilter && repo.ID != repoFilter {
			continue
		}

		if commits {
			count, err := gitscanner.IndexRepo(s, repo)
			if err != nil {
				fmt.Printf("  %-40s ERROR indexing commits: %v\n", repo.Name, err)
			} else {
				fmt.Printf("  %-40s %d commits indexed\n", repo.Name, count)
			}
		}

		remoteURL, err := gitscanner.GetRemoteURL(repo.Path)
		if err != nil {
			fmt.Printf("  %-40s SKIP forge (no origin remote)\n", repo.Name)
			continue
		}

		r, err := forge.ParseRemote(remoteURL)
		if err != nil {
			fmt.Printf("  %-40s SKIP forge (not a git remote)\n", repo.Name)
			continue
		}
		hk, known := hostKind(hostKinds, r.Host)
		switch {
		case !known:
			fmt.Printf("  %-40s SKIP forge (unknown host %s; add it under forges in config)\n", repo.Name, r.Host)
			continue
		case hk == "unsupported":
			fmt.Printf("  %-40s SKIP forge (unsupported forge %s)\n", repo.Name, r.Host)
			continue
		case !slices.Contains(kinds, hk):
			fmt.Printf("  %-40s SKIP forge (not a %s repo)\n", repo.Name, forgeLabels(kinds))
			continue
		}

		fmt.Printf("  %-40s syncing %s/%s/%s...\n", repo.Name, r.Host, r.Owner, r.Repo)
		issues, prs, runs, err := syncerFor(hk)(s, repo.ID, r, opts)
		if err != nil {
			fmt.Printf("  %-40s ERROR: %v\n", repo.Name, err)
			continue
		}
		fmt.Printf("  %-40s %d issues, %d %s, %d CI runs\n", repo.Name, issues, prs, prNoun(hk), runs)
	}

	return nil
}

func validForgeKind(kind string) bool {
	return kind == "github" || kind == "gitlab" || kind == "unsupported"
}

// classifyUnknownHosts asks, on an interactive terminal, which kind each
// unlisted forge host is, and saves the answers under forges in the config.
// Without a terminal the hosts stay unknown and their repos are skipped.
func classifyUnknownHosts(cfg *config.Config, kinds map[string]string, repoFilter string, repos []*model.Repository) (map[string]string, error) {
	var unknown []string
	seen := map[string]bool{}
	for _, repo := range repos {
		if repoFilter != "" && repo.Name != repoFilter && repo.ID != repoFilter {
			continue
		}
		remoteURL, err := gitscanner.GetRemoteURL(repo.Path)
		if err != nil {
			continue
		}
		r, err := forge.ParseRemote(remoteURL)
		if err != nil {
			continue
		}
		if _, known := hostKind(kinds, r.Host); known || seen[r.Host] {
			continue
		}
		seen[r.Host] = true
		unknown = append(unknown, r.Host)
	}

	if len(unknown) == 0 || !isTerminal() {
		return kinds, nil
	}

	in := bufio.NewReader(os.Stdin)
	for _, host := range unknown {
		kind := askForgeKind(in, host)
		if kind == "" {
			continue
		}
		kinds[host] = kind
		cfg.Forges = append(cfg.Forges, config.Forge{Host: host, Kind: kind})
	}
	if err := config.Save(cfg); err != nil {
		return kinds, fmt.Errorf("save forges: %w", err)
	}
	return kinds, nil
}

// askForgeKind prompts until the answer is a known kind. An empty answer skips
// the host for this run.
func askForgeKind(in *bufio.Reader, host string) string {
	for {
		fmt.Printf("Unknown forge host %s. What is it? [github / gitlab / unsupported / Enter to skip]: ", host)
		answer, err := in.ReadString('\n')
		answer = strings.ToLower(strings.TrimSpace(answer))
		if validForgeKind(answer) {
			return answer
		}
		if answer == "" || err != nil {
			return ""
		}
		fmt.Println("  please answer github, gitlab or unsupported")
	}
}

func isTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func forgeLabel(kind string) string {
	if kind == "gitlab" {
		return "GitLab"
	}
	return "GitHub"
}

// forgeLabels names the given kinds for a message, e.g. "GitHub or GitLab".
func forgeLabels(kinds []string) string {
	labels := make([]string, len(kinds))
	for i, k := range kinds {
		labels[i] = forgeLabel(k)
	}
	return strings.Join(labels, " or ")
}

func prNoun(kind string) string {
	if kind == "gitlab" {
		return "MRs"
	}
	return "PRs"
}

// addForgeSyncFlags registers the flags shared by the sync commands.
func addForgeSyncFlags(c *cobra.Command) {
	addScopeFlags(c)
	c.Flags().Bool("full", false, "Ignore the last sync watermark and fetch everything")
	c.Flags().Bool("with-comments", false, "Also fetch comments (slower: one request per changed item)")
}

package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/rlespinasse/ihatt/internal/config"
	"github.com/rlespinasse/ihatt/internal/store"
	"github.com/spf13/cobra"
)

var linksCmd = &cobra.Command{
	Use:   "links <repo-or-entity>",
	Short: "Show cross-references for a repository or entity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		s, err := store.NewBoltStore(config.DBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer s.Close()

		// Find the repo
		repos, err := s.ListRepos()
		if err != nil {
			return err
		}

		repoNames := make(map[string]string)
		for _, r := range repos {
			repoNames[r.ID] = r.Name
		}

		// Try to match as repo name or ID
		repoID := ""
		for _, r := range repos {
			if r.Name == target || r.ID == target {
				repoID = r.ID
				break
			}
		}

		if repoID == "" {
			return fmt.Errorf("repository not found: %s", target)
		}

		xrefs, err := s.GetCrossReferencesByRepo(repoID)
		if err != nil {
			return err
		}

		if len(xrefs) == 0 {
			fmt.Printf("No cross-references found for %s\n", target)
			return nil
		}

		fmt.Printf("Cross-references for %s (%d total)\n\n", target, len(xrefs))

		// References FROM this repo
		fmt.Println("--- References from this repo ---")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "SOURCE TYPE\tSOURCE ID\tTARGET REPO\tTARGET TYPE\t TARGET ID\n")
		fromCount := 0
		for _, xref := range xrefs {
			if xref.SourceRepoID == repoID {
				targetName := repoNames[xref.TargetRepoID]
				if targetName == "" {
					targetName = xref.TargetRepoID
				}
				sourceID := xref.SourceID
				if len(sourceID) > 8 {
					sourceID = sourceID[:8]
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t#%s\n",
					xref.SourceType, sourceID, targetName, xref.TargetType, xref.TargetID)
				fromCount++
			}
		}
		w.Flush()
		if fromCount == 0 {
			fmt.Println("  (none)")
		}

		fmt.Println()

		// References TO this repo
		fmt.Println("--- References to this repo ---")
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "SOURCE REPO\tSOURCE TYPE\tSOURCE ID\tTARGET TYPE\tTARGET ID\n")
		toCount := 0
		for _, xref := range xrefs {
			if xref.TargetRepoID == repoID && xref.SourceRepoID != repoID {
				sourceName := repoNames[xref.SourceRepoID]
				if sourceName == "" {
					sourceName = xref.SourceRepoID
				}
				sourceID := xref.SourceID
				if len(sourceID) > 8 {
					sourceID = sourceID[:8]
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t#%s\n",
					sourceName, xref.SourceType, sourceID, xref.TargetType, xref.TargetID)
				toCount++
			}
		}
		w.Flush()
		if toCount == 0 {
			fmt.Println("  (none)")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(linksCmd)
}

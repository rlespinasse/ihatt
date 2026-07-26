package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/rlespinasse/ihatt/internal/store"
)

type repoLink struct {
	FromName string
	ToName   string
	Count    int
}

type XRefGraphModel struct {
	store         store.Store
	width, height int
	links         []repoLink
	repos         []*model.Repository
	refCounts     map[string]int
	cursor        int
	loaded        bool
}

type xrefGraphLoadedMsg struct {
	links     []repoLink
	repos     []*model.Repository
	refCounts map[string]int
}

func NewXRefGraphModel(s store.Store) *XRefGraphModel {
	return &XRefGraphModel{store: s}
}

func (m *XRefGraphModel) Init() tea.Cmd {
	return func() tea.Msg {
		repos, _ := m.store.ListRepos()

		repoNames := make(map[string]string)
		for _, r := range repos {
			repoNames[r.ID] = r.Name
		}

		// Count links between repos
		linkCounts := make(map[string]int) // "from->to" -> count
		for _, repo := range repos {
			xrefs, _ := m.store.GetCrossReferencesByRepo(repo.ID)
			for _, xref := range xrefs {
				if xref.SourceRepoID != xref.TargetRepoID {
					fromName := repoNames[xref.SourceRepoID]
					if fromName == "" {
						fromName = xref.SourceRepoID
					}
					toName := repoNames[xref.TargetRepoID]
					if toName == "" {
						toName = xref.TargetRepoID
					}
					key := fromName + "->" + toName
					linkCounts[key]++
				}
			}
		}

		var links []repoLink
		seen := make(map[string]bool)
		for k, count := range linkCounts {
			if seen[k] {
				continue
			}
			seen[k] = true
			parts := strings.SplitN(k, "->", 2)
			links = append(links, repoLink{FromName: parts[0], ToName: parts[1], Count: count})
		}

		sort.Slice(links, func(i, j int) bool {
			return links[i].Count > links[j].Count
		})

		refCounts := make(map[string]int)
		var reposWithRefs []*model.Repository
		for _, repo := range repos {
			xrefs, _ := m.store.GetCrossReferencesByRepo(repo.ID)
			if len(xrefs) > 0 {
				refCounts[repo.ID] = len(xrefs)
				reposWithRefs = append(reposWithRefs, repo)
			}
		}

		return xrefGraphLoadedMsg{links: links, repos: reposWithRefs, refCounts: refCounts}
	}
}

func (m *XRefGraphModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case xrefGraphLoadedMsg:
		m.links = msg.links
		m.repos = msg.repos
		m.refCounts = msg.refCounts
		m.loaded = true
		return nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, Keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, Keys.Down):
			if m.cursor < len(m.repos)-1 {
				m.cursor++
			}
		case key.Matches(msg, Keys.Enter):
			if len(m.repos) > 0 {
				return func() tea.Msg {
					return NavigateToDetailMsg{RepoID: m.repos[m.cursor].ID}
				}
			}
		}
	}
	return nil
}

func (m *XRefGraphModel) View() string {
	if !m.loaded {
		return StyleMuted.Render("Loading...")
	}

	var sb strings.Builder

	sb.WriteString(StyleTitle.Render("Cross-Reference Graph"))
	sb.WriteString("\n")

	if len(m.links) == 0 {
		sb.WriteString(StyleMuted.Render("No cross-references found between projects."))
		sb.WriteString("\n")
		sb.WriteString(StyleMuted.Render("Run 'ihatt xref' to analyze commits for cross-references."))
		sb.WriteString("\n\n")
	} else {
		sb.WriteString(StyleSubtitle.Render("Project Links"))
		sb.WriteString("\n\n")

		for _, link := range m.links {
			bar := strings.Repeat("█", min(link.Count, m.width/3))
			sb.WriteString(fmt.Sprintf("  %s -> %s  %s %d\n",
				StyleRepoName.Render(link.FromName),
				StyleRepoName.Render(link.ToName),
				StyleSuccess.Render(bar),
				link.Count,
			))
		}
		sb.WriteString("\n")
	}

	// Repo list for navigation
	sb.WriteString(StyleSubtitle.Render("Repositories"))
	sb.WriteString("\n")
	for i, r := range m.repos {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%-30s %d references", prefix, r.Name, m.refCounts[r.ID])
		if i == m.cursor {
			sb.WriteString(StyleSelected.Render(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m *XRefGraphModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

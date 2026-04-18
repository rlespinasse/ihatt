package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/rlespinasse/ihatt/internal/store"
)

type DetailModel struct {
	store         store.Store
	width, height int
	repoID        string
	repo          *model.Repository
	commits       []*model.Commit
	ghItems       []*model.GitHubItem
	xrefs         []*model.CrossReference
	repoNames     map[string]string
	cursor        int
	scroll        int
	loaded        bool
}

type detailLoadedMsg struct {
	repo      *model.Repository
	commits   []*model.Commit
	ghItems   []*model.GitHubItem
	xrefs     []*model.CrossReference
	repoNames map[string]string
}

func NewDetailModel(s store.Store) *DetailModel {
	return &DetailModel{store: s}
}

func (m *DetailModel) SetRepoID(id string) {
	m.repoID = id
	m.loaded = false
}

func (m *DetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		repo, _ := m.store.GetRepo(m.repoID)
		commits, _ := m.store.GetCommitsByRepo(m.repoID, 50)
		ghItems, _ := m.store.GetGitHubItemsByRepo(m.repoID)
		xrefs, _ := m.store.GetCrossReferencesByRepo(m.repoID)

		repos, _ := m.store.ListRepos()
		repoNames := make(map[string]string)
		for _, r := range repos {
			repoNames[r.ID] = r.Name
		}

		return detailLoadedMsg{
			repo: repo, commits: commits,
			ghItems: ghItems, xrefs: xrefs, repoNames: repoNames,
		}
	}
}

func (m *DetailModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case detailLoadedMsg:
		m.repo = msg.repo
		m.commits = msg.commits
		m.ghItems = msg.ghItems
		m.xrefs = msg.xrefs
		m.repoNames = msg.repoNames
		m.loaded = true
		return nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, Keys.Up):
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.scroll {
					m.scroll = m.cursor
				}
			}
		case key.Matches(msg, Keys.Down):
			m.cursor++
			visibleRows := m.height - 6
			if m.cursor >= m.scroll+visibleRows {
				m.scroll = m.cursor - visibleRows + 1
			}
		}
	}
	return nil
}

func (m *DetailModel) View() string {
	if !m.loaded {
		return StyleMuted.Render("Loading...")
	}

	var sb strings.Builder

	name := "Unknown"
	if m.repo != nil {
		name = m.repo.Name
		sb.WriteString(StyleTitle.Render(name))
		sb.WriteString("\n")
		sb.WriteString(StyleMuted.Render(m.repo.Path))
		sb.WriteString("\n\n")
	}

	// Commits section
	sb.WriteString(StyleSubtitle.Render(fmt.Sprintf("Commits (%d)", len(m.commits))))
	sb.WriteString("\n")

	visibleRows := m.height - 10
	if visibleRows < 5 {
		visibleRows = 10
	}

	for i, c := range m.commits {
		if i < m.scroll {
			continue
		}
		if i-m.scroll >= visibleRows {
			break
		}

		msg := c.Message
		if idx := strings.Index(msg, "\n"); idx != -1 {
			msg = msg[:idx]
		}
		maxMsg := m.width - 35
		if maxMsg < 20 {
			maxMsg = 20
		}
		if len(msg) > maxMsg {
			msg = msg[:maxMsg-3] + "..."
		}

		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}

		line := fmt.Sprintf("%s%s %s %s %s",
			prefix,
			StyleDate.Render(c.Date.Format("2006-01-02 15:04")),
			StyleHash.Render(c.Hash[:7]),
			StyleAuthor.Render(c.Author),
			msg,
		)
		sb.WriteString(line + "\n")
	}

	// GitHub items
	if len(m.ghItems) > 0 {
		sb.WriteString("\n")
		sb.WriteString(StyleSubtitle.Render(fmt.Sprintf("GitHub (%d)", len(m.ghItems))))
		sb.WriteString("\n")

		for _, item := range m.ghItems {
			var style lipgloss.Style
			switch {
			case item.State == "merged":
				style = StyleMerged
			case item.Type == "pr":
				style = StylePR
			default:
				style = StyleIssue
			}

			title := item.Title
			if len(title) > 60 {
				title = title[:57] + "..."
			}

			typeLabel := strings.ToUpper(item.Type)
			sb.WriteString(fmt.Sprintf("  %s #%-4d [%s] %s\n",
				style.Render(typeLabel),
				item.Number,
				item.State,
				title,
			))
		}
	}

	// Cross-references
	if len(m.xrefs) > 0 {
		sb.WriteString("\n")
		sb.WriteString(StyleSubtitle.Render("Cross-References"))
		sb.WriteString("\n")

		for _, xref := range m.xrefs {
			if xref.SourceRepoID == m.repoID {
				targetName := m.repoNames[xref.TargetRepoID]
				if targetName == "" {
					targetName = xref.TargetRepoID
				}
				sb.WriteString(fmt.Sprintf("  -> %s %s #%s\n",
					StyleRepoName.Render(targetName),
					xref.TargetType,
					xref.TargetID,
				))
			} else {
				sourceName := m.repoNames[xref.SourceRepoID]
				if sourceName == "" {
					sourceName = xref.SourceRepoID
				}
				sb.WriteString(fmt.Sprintf("  <- %s %s %s\n",
					StyleRepoName.Render(sourceName),
					xref.SourceType,
					xref.SourceID[:8],
				))
			}
		}
	}

	return sb.String()
}

func (m *DetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

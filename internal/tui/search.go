package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/rlespinasse/ihatt/internal/store"
)

type SearchModel struct {
	store         store.Store
	width, height int
	input         textinput.Model
	results       []*model.Commit
	repoNames     map[string]string
	cursor        int
	focused       bool
	loaded        bool
}

type searchResultsMsg struct {
	results   []*model.Commit
	repoNames map[string]string
}

func NewSearchModel(s store.Store) *SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Search commits..."
	ti.CharLimit = 100
	ti.Width = 60

	return &SearchModel{
		store: s,
		input: ti,
	}
}

func (m *SearchModel) Init() tea.Cmd {
	m.input.Focus()
	m.focused = true
	m.loaded = true
	return textinput.Blink
}

func (m *SearchModel) IsInputFocused() bool {
	return m.focused
}

func (m *SearchModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case searchResultsMsg:
		m.results = msg.results
		m.repoNames = msg.repoNames
		return nil

	case tea.KeyMsg:
		if m.focused {
			switch msg.Type {
			case tea.KeyEnter:
				return m.doSearch()
			case tea.KeyEsc:
				m.focused = false
				m.input.Blur()
				return nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return cmd
		}

		// Navigate results
		switch {
		case key.Matches(msg, Keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, Keys.Down):
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
		case key.Matches(msg, Keys.Search):
			m.focused = true
			m.input.Focus()
			return textinput.Blink
		case key.Matches(msg, Keys.Enter):
			if len(m.results) > 0 {
				c := m.results[m.cursor]
				return func() tea.Msg {
					return NavigateToDetailMsg{RepoID: c.RepoID}
				}
			}
		}
	}
	return nil
}

func (m *SearchModel) doSearch() tea.Cmd {
	query := m.input.Value()
	if query == "" {
		return nil
	}
	m.focused = false
	m.input.Blur()

	return func() tea.Msg {
		results, _ := m.store.SearchCommits(query, nil, nil, "", "")

		repos, _ := m.store.ListRepos()
		repoNames := make(map[string]string)
		for _, r := range repos {
			repoNames[r.ID] = r.Name
		}

		return searchResultsMsg{results: results, repoNames: repoNames}
	}
}

func (m *SearchModel) View() string {
	var sb strings.Builder

	sb.WriteString(StyleTitle.Render("Search"))
	sb.WriteString("\n")
	sb.WriteString(m.input.View())
	sb.WriteString("\n\n")

	if len(m.results) == 0 && m.input.Value() != "" && !m.focused {
		sb.WriteString(StyleMuted.Render("No results found"))
		sb.WriteString("\n")
		return sb.String()
	}

	if len(m.results) > 0 {
		sb.WriteString(StyleMuted.Render(fmt.Sprintf("%d results", len(m.results))))
		sb.WriteString("\n\n")
	}

	visibleRows := m.height - 6
	if visibleRows < 5 {
		visibleRows = 10
	}

	scroll := 0
	if m.cursor >= visibleRows {
		scroll = m.cursor - visibleRows + 1
	}

	for i, c := range m.results {
		if i < scroll {
			continue
		}
		if i-scroll >= visibleRows {
			break
		}

		msg := c.Message
		if idx := strings.Index(msg, "\n"); idx != -1 {
			msg = msg[:idx]
		}
		maxMsg := m.width - 40
		if maxMsg < 20 {
			maxMsg = 20
		}
		if len(msg) > maxMsg {
			msg = msg[:maxMsg-3] + "..."
		}

		name := m.repoNames[c.RepoID]
		if name == "" {
			name = c.RepoID[:8]
		}

		prefix := "  "
		if i == m.cursor && !m.focused {
			prefix = "> "
		}

		line := fmt.Sprintf("%s%s %s %s %s %s",
			prefix,
			StyleDate.Render(c.Date.Format("2006-01-02")),
			StyleRepoName.Render(name),
			StyleHash.Render(c.Hash[:7]),
			StyleAuthor.Render(c.Author),
			msg,
		)
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

func (m *SearchModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

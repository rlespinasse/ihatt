package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/rlespinasse/ihatt/internal/store"
)

type DashboardModel struct {
	store         store.Store
	width, height int
	repos         []*model.Repository
	recentCommits []*model.Commit
	heatmap       map[string]int // date -> commit count
	selectedRepo  int
	loaded        bool
}

type dashboardLoadedMsg struct {
	repos   []*model.Repository
	commits []*model.Commit
	heatmap map[string]int
}

func NewDashboardModel(s store.Store) *DashboardModel {
	return &DashboardModel{store: s}
}

func (m *DashboardModel) Init() tea.Cmd {
	return func() tea.Msg {
		repos, _ := m.store.ListRepos()

		// Get last 90 days of commits for heatmap
		now := time.Now()
		from := now.AddDate(0, 0, -90)
		to := now
		commits, _ := m.store.GetCommitsByTimeRange(from, to)

		heatmap := make(map[string]int)
		for _, c := range commits {
			day := c.Date.Format("2006-01-02")
			heatmap[day]++
		}

		// Get last 15 commits
		var recent []*model.Commit
		if len(commits) > 15 {
			// Sort by date descending
			sort.Slice(commits, func(i, j int) bool {
				return commits[i].Date.After(commits[j].Date)
			})
			recent = commits[:15]
		} else {
			sort.Slice(commits, func(i, j int) bool {
				return commits[i].Date.After(commits[j].Date)
			})
			recent = commits
		}

		return dashboardLoadedMsg{repos: repos, commits: recent, heatmap: heatmap}
	}
}

func (m *DashboardModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case dashboardLoadedMsg:
		m.repos = msg.repos
		m.recentCommits = msg.commits
		m.heatmap = msg.heatmap
		m.loaded = true
		return nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, Keys.Up):
			if m.selectedRepo > 0 {
				m.selectedRepo--
			}
		case key.Matches(msg, Keys.Down):
			if m.selectedRepo < len(m.repos)-1 {
				m.selectedRepo++
			}
		case key.Matches(msg, Keys.Enter):
			if len(m.repos) > 0 {
				return func() tea.Msg {
					return NavigateToDetailMsg{RepoID: m.repos[m.selectedRepo].ID}
				}
			}
		case key.Matches(msg, Keys.Refresh):
			return m.Init()
		}
	}
	return nil
}

func (m *DashboardModel) View() string {
	if !m.loaded {
		return StyleMuted.Render("Loading...")
	}

	var sections []string

	// Title
	sections = append(sections, StyleTitle.Render("Dashboard"))

	// Activity heatmap (last 12 weeks)
	sections = append(sections, m.renderHeatmap())

	// Today's summary
	sections = append(sections, m.renderTodaySummary())

	// Recent commits
	sections = append(sections, m.renderRecentCommits())

	// Repository list
	sections = append(sections, m.renderRepoList())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *DashboardModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *DashboardModel) renderHeatmap() string {
	var sb strings.Builder
	sb.WriteString(StyleSubtitle.Render("Activity (last 12 weeks)"))
	sb.WriteString("\n")

	now := time.Now()
	blocks := []string{"░", "▒", "▓", "█"}

	for week := 11; week >= 0; week-- {
		for day := 0; day < 7; day++ {
			d := now.AddDate(0, 0, -(week*7 + (6 - day)))
			key := d.Format("2006-01-02")
			count := m.heatmap[key]

			var block string
			switch {
			case count == 0:
				block = StyleMuted.Render("·")
			case count <= 3:
				block = StyleSuccess.Render(blocks[0])
			case count <= 8:
				block = StyleSuccess.Render(blocks[1])
			case count <= 15:
				block = StyleSuccess.Render(blocks[2])
			default:
				block = StyleSuccess.Render(blocks[3])
			}
			sb.WriteString(block)
		}
		sb.WriteString(" ")
	}
	sb.WriteString("\n")
	return sb.String()
}

func (m *DashboardModel) renderTodaySummary() string {
	today := time.Now().Format("2006-01-02")
	count := m.heatmap[today]
	return StyleSubtitle.Render("Today") + " " +
		StyleMuted.Render(fmt.Sprintf("%d commits", count)) + "\n"
}

func (m *DashboardModel) renderRecentCommits() string {
	var sb strings.Builder
	sb.WriteString(StyleSubtitle.Render("Recent Activity"))
	sb.WriteString("\n")

	if len(m.recentCommits) == 0 {
		sb.WriteString(StyleMuted.Render("  No recent commits\n"))
		return sb.String()
	}

	repoNames := make(map[string]string)
	for _, r := range m.repos {
		repoNames[r.ID] = r.Name
	}

	for _, c := range m.recentCommits {
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

		name := repoNames[c.RepoID]
		if name == "" {
			name = c.RepoID[:8]
		}

		line := fmt.Sprintf("  %s %s %s %s",
			StyleDate.Render(c.Date.Format("Jan 02 15:04")),
			StyleRepoName.Render(name),
			StyleHash.Render(c.Hash[:7]),
			msg,
		)
		sb.WriteString(line + "\n")
	}
	return sb.String()
}

func (m *DashboardModel) renderRepoList() string {
	var sb strings.Builder
	sb.WriteString(StyleSubtitle.Render("Repositories"))
	sb.WriteString("\n")

	if len(m.repos) == 0 {
		sb.WriteString(StyleMuted.Render("  No repositories tracked\n"))
		return sb.String()
	}

	for i, r := range m.repos {
		count, _ := m.store.CountCommitsByRepo(r.ID)
		line := fmt.Sprintf("  %-30s %d commits", r.Name, count)

		if i == m.selectedRepo {
			sb.WriteString(StyleSelected.Render(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

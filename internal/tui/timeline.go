package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/rlespinasse/ihatt/internal/store"
)

type dayGroup struct {
	Date     string
	Commits  []*model.Commit
	Expanded bool
}

type TimelineModel struct {
	store         store.Store
	width, height int
	days          []dayGroup
	cursor        int
	scroll        int
	repoNames     map[string]string
	loaded        bool
}

type timelineLoadedMsg struct {
	days      []dayGroup
	repoNames map[string]string
}

func NewTimelineModel(s store.Store) *TimelineModel {
	return &TimelineModel{store: s}
}

func (m *TimelineModel) Init() tea.Cmd {
	return func() tea.Msg {
		now := time.Now()
		from := now.AddDate(0, 0, -30)
		to := now
		commits, _ := m.store.GetCommitsByTimeRange(from, to)

		repos, _ := m.store.ListRepos()
		repoNames := make(map[string]string)
		for _, r := range repos {
			repoNames[r.ID] = r.Name
		}

		// Group by day
		byDay := make(map[string][]*model.Commit)
		for _, c := range commits {
			day := c.Date.Format("2006-01-02")
			byDay[day] = append(byDay[day], c)
		}

		// Sort days descending
		dayKeys := make([]string, 0, len(byDay))
		for k := range byDay {
			dayKeys = append(dayKeys, k)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(dayKeys)))

		days := make([]dayGroup, len(dayKeys))
		for i, k := range dayKeys {
			cs := byDay[k]
			sort.Slice(cs, func(a, b int) bool {
				return cs[a].Date.After(cs[b].Date)
			})
			days[i] = dayGroup{Date: k, Commits: cs}
		}

		return timelineLoadedMsg{days: days, repoNames: repoNames}
	}
}

func (m *TimelineModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case timelineLoadedMsg:
		m.days = msg.days
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
			if m.cursor < len(m.days)-1 {
				m.cursor++
				visibleRows := m.height - 4
				if m.cursor >= m.scroll+visibleRows {
					m.scroll = m.cursor - visibleRows + 1
				}
			}
		case key.Matches(msg, Keys.Enter):
			if len(m.days) > 0 {
				m.days[m.cursor].Expanded = !m.days[m.cursor].Expanded
			}
		}
	}
	return nil
}

func (m *TimelineModel) View() string {
	if !m.loaded {
		return StyleMuted.Render("Loading...")
	}

	var sb strings.Builder
	sb.WriteString(StyleTitle.Render("Timeline (last 30 days)"))
	sb.WriteString("\n")

	if len(m.days) == 0 {
		sb.WriteString(StyleMuted.Render("No activity found\n"))
		return sb.String()
	}

	visibleRows := m.height - 4
	if visibleRows < 1 {
		visibleRows = 10
	}

	lines := 0
	for i, day := range m.days {
		if i < m.scroll {
			continue
		}
		if lines >= visibleRows {
			break
		}

		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}

		indicator := "+"
		if day.Expanded {
			indicator = "-"
		}

		header := fmt.Sprintf("%s%s %s  %s",
			prefix,
			indicator,
			StyleSubtitle.Render(day.Date),
			StyleMuted.Render(fmt.Sprintf("(%d commits)", len(day.Commits))),
		)
		sb.WriteString(header + "\n")
		lines++

		if day.Expanded {
			// Group by repo within day
			byRepo := make(map[string][]*model.Commit)
			for _, c := range day.Commits {
				byRepo[c.RepoID] = append(byRepo[c.RepoID], c)
			}

			for repoID, commits := range byRepo {
				name := m.repoNames[repoID]
				if name == "" {
					name = repoID[:8]
				}
				sb.WriteString(fmt.Sprintf("    %s\n", StyleRepoName.Render(name)))
				lines++

				for _, c := range commits {
					if lines >= visibleRows {
						break
					}
					msg := c.Message
					if idx := strings.Index(msg, "\n"); idx != -1 {
						msg = msg[:idx]
					}
					maxMsg := m.width - 30
					if maxMsg < 20 {
						maxMsg = 20
					}
					if len(msg) > maxMsg {
						msg = msg[:maxMsg-3] + "..."
					}
					sb.WriteString(fmt.Sprintf("      %s %s %s\n",
						StyleDate.Render(c.Date.Format("15:04")),
						StyleHash.Render(c.Hash[:7]),
						msg,
					))
					lines++
				}
			}
		}
	}

	return sb.String()
}

func (m *TimelineModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

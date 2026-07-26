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

type timelineRowKind int

const (
	rowDay timelineRowKind = iota
	rowRepo
	rowCommit
)

type timelineRow struct {
	kind   timelineRowKind
	dayIdx int
	repoID string
	commit *model.Commit
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

// buildRows flattens the day/repo/commit tree into the rows currently
// visible given each day's expanded state, so cursor and scroll can operate
// on the same units as rendering.
func (m *TimelineModel) buildRows() []timelineRow {
	var rows []timelineRow
	for di, day := range m.days {
		rows = append(rows, timelineRow{kind: rowDay, dayIdx: di})
		if !day.Expanded {
			continue
		}

		byRepo := make(map[string][]*model.Commit)
		var repoOrder []string
		for _, c := range day.Commits {
			if _, ok := byRepo[c.RepoID]; !ok {
				repoOrder = append(repoOrder, c.RepoID)
			}
			byRepo[c.RepoID] = append(byRepo[c.RepoID], c)
		}

		for _, repoID := range repoOrder {
			rows = append(rows, timelineRow{kind: rowRepo, dayIdx: di, repoID: repoID})
			for _, c := range byRepo[repoID] {
				rows = append(rows, timelineRow{kind: rowCommit, dayIdx: di, repoID: repoID, commit: c})
			}
		}
	}
	return rows
}

func (m *TimelineModel) visibleRows() int {
	visibleRows := m.height - 4
	if visibleRows < 1 {
		visibleRows = 10
	}
	return visibleRows
}

func (m *TimelineModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case timelineLoadedMsg:
		m.days = msg.days
		m.repoNames = msg.repoNames
		m.loaded = true
		m.cursor = 0
		m.scroll = 0
		return nil

	case tea.KeyMsg:
		rows := m.buildRows()
		switch {
		case key.Matches(msg, Keys.Up):
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.scroll {
					m.scroll = m.cursor
				}
			}
		case key.Matches(msg, Keys.Down):
			if m.cursor < len(rows)-1 {
				m.cursor++
				visibleRows := m.visibleRows()
				if m.cursor >= m.scroll+visibleRows {
					m.scroll = m.cursor - visibleRows + 1
				}
			}
		case key.Matches(msg, Keys.Enter):
			if m.cursor < 0 || m.cursor >= len(rows) {
				break
			}
			row := rows[m.cursor]
			switch row.kind {
			case rowDay:
				m.days[row.dayIdx].Expanded = !m.days[row.dayIdx].Expanded

				newRows := m.buildRows()
				if m.cursor >= len(newRows) {
					m.cursor = len(newRows) - 1
				}
				if m.scroll > m.cursor {
					m.scroll = m.cursor
				}
			case rowCommit:
				c := row.commit
				name := m.repoNames[c.RepoID]
				if name == "" {
					name = c.RepoID
				}
				return func() tea.Msg {
					return NavigateToCommitDetailMsg{Commit: c, RepoName: name}
				}
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

	rows := m.buildRows()
	visibleRows := m.visibleRows()

	lines := 0
	for i := m.scroll; i < len(rows) && lines < visibleRows; i++ {
		row := rows[i]

		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}

		switch row.kind {
		case rowDay:
			day := m.days[row.dayIdx]
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

		case rowRepo:
			name := m.repoNames[row.repoID]
			if name == "" {
				name = row.repoID[:8]
			}
			sb.WriteString(fmt.Sprintf("%s  %s\n", prefix, StyleRepoName.Render(name)))

		case rowCommit:
			c := row.commit
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
			sb.WriteString(fmt.Sprintf("%s    %s %s %s\n",
				prefix,
				StyleDate.Render(c.Date.Format("15:04")),
				StyleHash.Render(c.Hash[:7]),
				msg,
			))
		}
		lines++
	}

	return sb.String()
}

func (m *TimelineModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

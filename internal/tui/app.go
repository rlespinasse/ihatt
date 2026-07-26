package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rlespinasse/ihatt/internal/store"
)

type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenTimeline
	ScreenDetail
	ScreenSearch
	ScreenXRefGraph
	ScreenCommitDetail
)

type App struct {
	store         store.Store
	screen        Screen
	prevScreen    Screen
	width, height int

	dashboard    *DashboardModel
	timeline     *TimelineModel
	detail       *DetailModel
	search       *SearchModel
	xrefgraph    *XRefGraphModel
	commitDetail *CommitDetailModel
}

func NewApp(s store.Store) *App {
	return &App{
		store:        s,
		screen:       ScreenDashboard,
		dashboard:    NewDashboardModel(s),
		timeline:     NewTimelineModel(s),
		detail:       NewDetailModel(s),
		search:       NewSearchModel(s),
		xrefgraph:    NewXRefGraphModel(s),
		commitDetail: NewCommitDetailModel(),
	}
}

func (a *App) Init() tea.Cmd {
	return a.dashboard.Init()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.dashboard.SetSize(msg.Width, msg.Height-2)
		a.timeline.SetSize(msg.Width, msg.Height-2)
		a.detail.SetSize(msg.Width, msg.Height-2)
		a.search.SetSize(msg.Width, msg.Height-2)
		a.xrefgraph.SetSize(msg.Width, msg.Height-2)
		a.commitDetail.SetSize(msg.Width, msg.Height-2)
		return a, nil

	case tea.KeyMsg:
		// Global keys
		if key.Matches(msg, Keys.Quit) {
			if a.screen == ScreenSearch && a.search.IsInputFocused() {
				// Let search handle it
			} else {
				return a, tea.Quit
			}
		}

		// Navigation
		if a.screen != ScreenSearch || !a.search.IsInputFocused() {
			switch {
			case key.Matches(msg, Keys.Dashboard):
				a.switchScreen(ScreenDashboard)
				return a, a.dashboard.Init()
			case key.Matches(msg, Keys.Timeline):
				a.switchScreen(ScreenTimeline)
				return a, a.timeline.Init()
			case key.Matches(msg, Keys.Search):
				a.switchScreen(ScreenSearch)
				return a, a.search.Init()
			case key.Matches(msg, Keys.Links):
				a.switchScreen(ScreenXRefGraph)
				return a, a.xrefgraph.Init()
			case key.Matches(msg, Keys.Back):
				if a.screen != ScreenDashboard {
					a.screen = a.prevScreen
					return a, nil
				}
			}
		}

	case NavigateToDetailMsg:
		a.detail.SetRepoID(msg.RepoID)
		a.switchScreen(ScreenDetail)
		return a, a.detail.Init()

	case NavigateToCommitDetailMsg:
		a.commitDetail.SetCommit(msg.Commit, msg.RepoName)
		a.switchScreen(ScreenCommitDetail)
		return a, nil
	}

	// Delegate to active screen
	var cmd tea.Cmd
	switch a.screen {
	case ScreenDashboard:
		cmd = a.dashboard.Update(msg)
	case ScreenTimeline:
		cmd = a.timeline.Update(msg)
	case ScreenDetail:
		cmd = a.detail.Update(msg)
	case ScreenSearch:
		cmd = a.search.Update(msg)
	case ScreenXRefGraph:
		cmd = a.xrefgraph.Update(msg)
	case ScreenCommitDetail:
		cmd = a.commitDetail.Update(msg)
	}
	return a, cmd
}

func (a *App) View() string {
	var content string
	switch a.screen {
	case ScreenDashboard:
		content = a.dashboard.View()
	case ScreenTimeline:
		content = a.timeline.View()
	case ScreenDetail:
		content = a.detail.View()
	case ScreenSearch:
		content = a.search.View()
	case ScreenXRefGraph:
		content = a.xrefgraph.View()
	case ScreenCommitDetail:
		content = a.commitDetail.View()
	}

	// Status bar
	statusItems := []string{"d:dashboard", "t:timeline", "/:search", "l:links", "q:quit"}
	status := StyleStatusBar.Width(a.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Top,
			StyleTitle.Render("ihatt"),
			"  ",
			StyleMuted.Render(joinWith(statusItems, " | ")),
		),
	)

	return lipgloss.JoinVertical(lipgloss.Left, content, status)
}

func (a *App) switchScreen(s Screen) {
	a.prevScreen = a.screen
	a.screen = s
}

func joinWith(items []string, sep string) string {
	result := ""
	for i, item := range items {
		if i > 0 {
			result += sep
		}
		result += item
	}
	return result
}

// NavigateToDetailMsg triggers navigation to the detail screen for a repo.
type NavigateToDetailMsg struct {
	RepoID string
}

package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rlespinasse/ihatt/internal/model"
)

// NavigateToCommitDetailMsg triggers navigation to the commit detail screen.
type NavigateToCommitDetailMsg struct {
	Commit   *model.Commit
	RepoName string
}

type CommitDetailModel struct {
	width, height int
	commit        *model.Commit
	repoName      string
}

func NewCommitDetailModel() *CommitDetailModel {
	return &CommitDetailModel{}
}

func (m *CommitDetailModel) SetCommit(c *model.Commit, repoName string) {
	m.commit = c
	m.repoName = repoName
}

func (m *CommitDetailModel) Init() tea.Cmd {
	return nil
}

func (m *CommitDetailModel) Update(msg tea.Msg) tea.Cmd {
	return nil
}

func (m *CommitDetailModel) View() string {
	if m.commit == nil {
		return StyleMuted.Render("No commit selected")
	}
	c := m.commit

	var sb strings.Builder
	sb.WriteString(StyleTitle.Render("Commit " + c.Hash[:7]))
	sb.WriteString("\n\n")

	if m.repoName != "" {
		sb.WriteString(fmt.Sprintf("%s %s\n", StyleSubtitle.Render("Repo:"), StyleRepoName.Render(m.repoName)))
	}
	sb.WriteString(fmt.Sprintf("%s   %s\n", StyleSubtitle.Render("Hash:"), StyleHash.Render(c.Hash)))
	sb.WriteString(fmt.Sprintf("%s %s <%s>\n", StyleSubtitle.Render("Author:"), StyleAuthor.Render(c.Author), c.Email))
	sb.WriteString(fmt.Sprintf("%s   %s\n", StyleSubtitle.Render("Date:"), StyleDate.Render(c.Date.Format("2006-01-02 15:04:05"))))
	sb.WriteString(fmt.Sprintf("%s %s\n", StyleSubtitle.Render("Files:"),
		StyleMuted.Render(fmt.Sprintf("+%d added, ~%d modified, -%d deleted", c.FilesAdded, c.FilesModified, c.FilesDeleted))))

	sb.WriteString("\n")
	sb.WriteString(StyleSubtitle.Render("Message:"))
	sb.WriteString("\n")
	for _, line := range strings.Split(c.Message, "\n") {
		sb.WriteString("  " + line + "\n")
	}

	return sb.String()
}

func (m *CommitDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/user/ws/pkg/types"
)

type Model struct {
	table      table.Model
	viewport   viewport.Model
	results    []types.Result
	command    string
	workspace  string
	startTime  time.Time
	failReason string
	mode       string
	width      int
	height     int
	detailView bool
	detailIdx  int
}

func NewModel(command, workspace, mode string) Model {
	columns := []table.Column{
		{Title: "Repo", Width: 20},
		{Title: "Group", Width: 12},
		{Title: "Status", Width: 10},
		{Title: "Time", Width: 10},
		{Title: "Detail", Width: 40},
	}
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(false),
		table.WithHeight(10),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#6B7280")).
		BorderBottom(true).
		Bold(false)
	t.SetStyles(s)

	return Model{
		table:     t,
		viewport:  viewport.New(80, 20),
		command:   command,
		workspace: workspace,
		startTime: time.Now(),
		mode:      mode,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(msg.Width - 4)
		m.table.SetHeight(msg.Height - 6)
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 6
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "r":
			if m.mode == "monitor" {
				return m, nil // refresh handled by caller
			}
		case "enter":
			if !m.detailView && len(m.results) > 0 {
				m.detailView = true
				if m.table.Cursor() < len(m.results) {
					m.detailIdx = m.table.Cursor()
				}
				r := m.results[m.detailIdx]
				m.viewport.SetContent(fmt.Sprintf(
					"Repo: %s\nStatus: %s\nDetail:\n%s\nStderr:\n%s",
					r.RepoName, r.Status, r.Detail, r.ErrorStr,
				))
			} else {
				m.detailView = false
			}
		}

	case TickMsg:
		m.results = msg.Results
		if msg.Complete && m.mode == "auto" {
			return m, tea.Quit
		}
	}

	m.updateRows()
	return m, nil
}

func (m *Model) updateRows() {
	var rows []table.Row
	for _, r := range m.results {
		icon := StatusIcon(string(r.Status))
		dur := "—"
		if r.Duration > 0 {
			dur = r.Duration.Truncate(time.Millisecond).String()
		}
		detail := r.Detail
		if len(detail) > 40 {
			detail = detail[:37] + "..."
		}
		rows = append(rows, table.Row{
			r.RepoName,
			r.Group,
			icon + " " + string(r.Status),
			dur,
			detail,
		})
	}
	m.table.SetRows(rows)
}

func (m Model) View() string {
	if m.detailView {
		return m.viewport.View() + "\n" + HelpStyle.Render("Enter: back • q: quit")
	}

	elapsed := time.Since(m.startTime).Truncate(time.Second)
	completed := 0
	for _, r := range m.results {
		if r.Status != types.StatusPending && r.Status != types.StatusRunning {
			completed++
		}
	}

	header := TitleStyle.Render(fmt.Sprintf(
		"ws %s — %s — %d/%d completed — %s elapsed",
		m.command, m.workspace, completed, len(m.results), elapsed,
	))

	summary := types.Summarize(m.results)
	footer := FooterStyle.Render(fmt.Sprintf(
		"Results: %d passed | %d failed | %d cancelled | %d skipped | %d total",
		summary.Success, summary.Failed, summary.Cancelled, summary.Skipped, summary.Total,
	))

	help := HelpStyle.Render("q: quit • ↑↓: scroll • Enter: details")
	if m.mode == "monitor" {
		help = HelpStyle.Render("q: quit • r: refresh • ↑↓: scroll • Enter: details")
	}

	return strings.Join([]string{header, m.table.View(), footer, help}, "\n")
}

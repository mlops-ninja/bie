package main

import (
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ProgressMsg int64

const (
	padding  = 2
	maxWidth = 80
)

var statusBarStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
	Background(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#353533"}).Align(lipgloss.Left)

type Model struct {
	FilePath string
	Command  string
	FileSize int64
	Uploaded int64

	// For the progress bar
	progressWidth int

	width  int
	height int
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Cool, what was the actual key pressed?
		switch msg.String() {
		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case ProgressMsg:
		m.Uploaded += int64(msg)
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

		m.progressWidth = msg.Width - padding*2 - 4
		if m.progressWidth > maxWidth {
			m.progressWidth = maxWidth
		}
	default:
		// Do nothing
	}
	return m, nil
}

func (m Model) View() string {
	// header := lipgloss.NewStyle().
	// 	Align(lipgloss.Center).
	// 	Width(m.width).
	// 	Border(lipgloss.NormalBorder(), false, false, true, false).
	// 	Render("BIE")

	// footer := statusBarStyle.Width(m.width).Render("Press 'q' or 'ctrl+c' to exit | Press 'c' to show command for CURL")

	// pad := strings.Repeat(" ", padding)
	progressMdl := progress.New()
	progressMdl.Width = m.progressWidth
	progressMdl.SetPercent(float64(m.Uploaded) / float64(m.FileSize))
	// progress := pad + progressMdl.View() + "\n"

	// content := lipgloss.NewStyle().Height(m.height - lipgloss.Height(header) - lipgloss.Height(footer) - lipgloss.Height(progress)).Render("\n\nIn order to upload a file into " + m.FilePath + " run the following command:\n\n\n\n" + m.Command)
	content := "\n\nIn order to upload a file into " + m.FilePath + " run the following command:\n" + m.Command

	return content

	// return header + "\n" + content + "\n" + progress + "\n" + footer
}

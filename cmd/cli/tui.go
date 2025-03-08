package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type ProgressMsg int64

const (
	padding  = 2
	maxWidth = 80
)

type Model struct {
	FilePath string
	Command  string
	FileSize int64
	Uploaded int64

	// For the progress bar
	progressWidth int
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ProgressMsg:
		m.Uploaded += int64(msg)
	case tea.WindowSizeMsg:
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
	pad := strings.Repeat(" ", padding)
	progressMdl := progress.New()
	progressMdl.Width = m.progressWidth
	progressMdl.SetPercent(float64(m.Uploaded) / float64(m.FileSize))
	return "BIE:\nIn order to upload a file into " + m.FilePath + " run the following command:\n" + m.Command + "\n\n" + pad + progressMdl.View() + "\n"
}

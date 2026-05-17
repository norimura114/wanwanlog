package tui

import "github.com/charmbracelet/lipgloss"

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Reverse(true)

	filterStyle = lipgloss.NewStyle().
			Padding(0, 1)

	listPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	detailPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	selectedLineStyle = lipgloss.NewStyle().
				Bold(true)

	footerStyle = lipgloss.NewStyle().
			Reverse(true)

	footerErrorStyle = lipgloss.NewStyle().
				Reverse(true).
				Bold(true)

	helpStyle = lipgloss.NewStyle().
			Padding(1, 2)
)

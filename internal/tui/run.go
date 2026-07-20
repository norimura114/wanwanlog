package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

func Run(
	ctx context.Context,
	search SearchFunc,
	initial SearchResult,
	options ModelOptions,
) error {
	model := NewModel(ctx, search, initial.Records, options)
	model.total = initial.Total

	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	_, err := program.Run()
	return err
}

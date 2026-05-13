package tui

import (
	"context"
	"wanwanlog/internal/logrecord"

	tea "github.com/charmbracelet/bubbletea"
)

func Run(
	ctx context.Context,
	search SearchFunc,
	initialRecords []logrecord.LogRecord,
) error {
	model := NewModel(ctx, search, initialRecords)

	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	_, err := program.Run()
	return err
}

package tui

import (
	"context"
	"wanwanlog/internal/logrecord"

	"github.com/charmbracelet/bubbles/textinput"
)

type Mode int

const (
	BrowseMode Mode = iota
	FilterMode
	DetailMode
	HelpMode
)

type SearchFunc func(ctx context.Context, query string) ([]logrecord.LogRecord, error)

type Model struct {
	ctx context.Context

	mode Mode

	filterInput textinput.Model
	filterText  string

	records       []logrecord.LogRecord
	selectedIndex int
	detailOffset  int

	width  int
	height int

	err     error
	loading bool

	search SearchFunc
}

func NewModel(ctx context.Context, search SearchFunc, initialRecords []logrecord.LogRecord) Model {
	input := textinput.New()
	input.Placeholder = "service:api status:error @http.status >= 400"
	input.Prompt = "> "
	input.CharLimit = 512

	return Model{
		ctx:           ctx,
		mode:          BrowseMode,
		filterInput:   input,
		records:       initialRecords,
		selectedIndex: 0,
		search:        search,
	}
}

func (m Model) selectedRecord() *logrecord.LogRecord {
	if len(m.records) == 0 {
		return nil
	}

	if m.selectedIndex < 0 || m.selectedIndex >= len(m.records) {
		return nil
	}

	return &m.records[m.selectedIndex]
}

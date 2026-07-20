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
	ColumnsMode
)

type SearchFunc func(ctx context.Context, query string) ([]logrecord.LogRecord, error)

type ModelOptions struct {
	AttributeNames  []string
	SelectedColumns []string
	InitialQuery    string
}

type Model struct {
	ctx context.Context

	mode Mode

	filterInput textinput.Model
	filterText  string

	records       []logrecord.LogRecord
	selectedIndex int
	detailOffset  int
	detailRaw     bool

	columns      []Column
	columnDraft  []Column
	columnCursor int
	columnOffset int

	width  int
	height int

	err     error
	loading bool

	search SearchFunc
}

func NewModel(ctx context.Context, search SearchFunc, initialRecords []logrecord.LogRecord, options ...ModelOptions) Model {
	input := textinput.New()
	input.Placeholder = "service:api status:error @http.status >= 400"
	input.Prompt = "> "
	input.CharLimit = 512

	var option ModelOptions
	if len(options) > 0 {
		option = options[0]
	}

	return Model{
		ctx:           ctx,
		mode:          BrowseMode,
		filterInput:   input,
		filterText:    option.InitialQuery,
		records:       initialRecords,
		selectedIndex: 0,
		columns:       NewColumns(option.AttributeNames, option.SelectedColumns),
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

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

type SearchResult struct {
	Records []logrecord.LogRecord
	Total   int
}

type SearchFunc func(ctx context.Context, query string) (SearchResult, error)

type FollowEvent struct {
	AddedCount      int
	ParseErrorCount int
	Reset           bool
	AttributeNames  []string
	State           string
	Err             error
}

type ModelOptions struct {
	AttributeNames  []string
	SelectedColumns []string
	InitialQuery    string
	FollowEvents    <-chan FollowEvent
	Live            bool
	ParseErrorCount int
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

	err          error
	loading      bool
	total        int
	state        string
	parseErrors  int
	followEvents <-chan FollowEvent
	generation   uint64
	searching    bool
	dirty        bool

	search SearchFunc
}

func NewModel(ctx context.Context, search any, initialRecords []logrecord.LogRecord, options ...ModelOptions) Model {
	input := textinput.New()
	input.Placeholder = "service:api status:error @http.status >= 400"
	input.Prompt = "> "
	input.CharLimit = 512

	var option ModelOptions
	if len(options) > 0 {
		option = options[0]
	}

	var searchFunc SearchFunc
	switch fn := search.(type) {
	case SearchFunc:
		searchFunc = fn
	case func(context.Context, string) (SearchResult, error):
		searchFunc = fn
	case func(context.Context, string) ([]logrecord.LogRecord, error):
		searchFunc = func(ctx context.Context, query string) (SearchResult, error) {
			records, err := fn(ctx, query)
			return SearchResult{Records: records, Total: len(records)}, err
		}
	case nil:
	}
	state := ""
	if option.Live {
		state = "live"
	}
	return Model{
		ctx:           ctx,
		mode:          BrowseMode,
		filterInput:   input,
		filterText:    option.InitialQuery,
		records:       initialRecords,
		selectedIndex: 0,
		columns:       NewColumns(option.AttributeNames, option.SelectedColumns),
		total:         len(initialRecords),
		search:        searchFunc,
		followEvents:  option.FollowEvents,
		state:         state,
		parseErrors:   option.ParseErrorCount,
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

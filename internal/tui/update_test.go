package tui

import (
	"context"
	"testing"
	"wanwanlog/internal/logrecord"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModelKeepsInitialQuery(t *testing.T) {
	model := NewModel(
		context.Background(),
		func(context.Context, string) ([]logrecord.LogRecord, error) { return nil, nil },
		nil,
		ModelOptions{InitialQuery: "status:error"},
	)

	if model.filterText != "status:error" {
		t.Fatalf("filterText = %q, want status:error", model.filterText)
	}
}

func TestDetailScrollStopsWhenLastLineReachesViewportBottom(t *testing.T) {
	model := NewModel(t.Context(), nil, []logrecord.LogRecord{testLogRecord()})
	model.width = 80
	model.height = 24
	model.mode = DetailMode

	maxOffset := model.maxDetailOffset()
	model.detailOffset = maxOffset

	updated, _ := model.handleDetailKey(keyMsg("j"))
	got := updated.(Model)

	if got.detailOffset != maxOffset {
		t.Fatalf("detail offset = %d, want viewport-bottom offset %d", got.detailOffset, maxOffset)
	}

	rightWidth := model.width - model.width/2
	contentWidth := paneContentWidth(detailPaneStyle, rightWidth)
	bodyHeight := max(5, model.height-6)
	contentHeight := paneContentHeight(detailPaneStyle, bodyHeight)
	lineCount := len(formatDetail(testLogRecord(), contentWidth, false))
	if maxOffset != max(0, lineCount-contentHeight) {
		t.Fatalf(
			"maximum offset = %d, want line count %d - viewport height %d",
			maxOffset,
			lineCount,
			contentHeight,
		)
	}
}

func TestDetailRTogglesAttributesAndRaw(t *testing.T) {
	model := NewModel(t.Context(), nil, []logrecord.LogRecord{testLogRecord()})
	model.width = 80
	model.height = 24
	model.mode = DetailMode

	updated, _ := model.handleDetailKey(keyMsg("r"))
	raw := updated.(Model)
	if !raw.detailRaw {
		t.Fatal("r should switch detail to raw view")
	}

	updated, _ = raw.handleDetailKey(keyMsg("r"))
	attributes := updated.(Model)
	if attributes.detailRaw {
		t.Fatal("second r should switch detail back to attributes view")
	}
}

func TestOpeningAnotherDetailPreservesAndClampsScrollOffset(t *testing.T) {
	longRecord := testLogRecord()
	shortRecord := testLogRecord()
	shortRecord.Attributes = nil
	shortRecord.Raw = "{}"

	model := NewModel(t.Context(), nil, []logrecord.LogRecord{longRecord, longRecord, shortRecord})
	model.width = 80
	model.height = 24
	model.detailOffset = 5

	model.selectedIndex = 1
	updated, _ := model.handleBrowseKey(keyMsg("enter"))
	sameLength := updated.(Model)
	if sameLength.detailOffset != 5 {
		t.Fatalf("offset for same-length detail = %d, want 5", sameLength.detailOffset)
	}

	sameLength.mode = BrowseMode
	sameLength.selectedIndex = 2
	sameLength.detailOffset = sameLength.maxDetailOffset() + 10
	wantEndOffset := Model{
		records:       sameLength.records,
		selectedIndex: sameLength.selectedIndex,
		width:         sameLength.width,
		height:        sameLength.height,
	}.maxDetailOffset()

	updated, _ = sameLength.handleBrowseKey(keyMsg("enter"))
	shorter := updated.(Model)
	if shorter.detailOffset != wantEndOffset {
		t.Fatalf("offset for shorter detail = %d, want end offset %d", shorter.detailOffset, wantEndOffset)
	}
}

func TestHLKeysSwitchPanes(t *testing.T) {
	model := NewModel(t.Context(), nil, []logrecord.LogRecord{testLogRecord()})
	model.width = 80
	model.height = 24

	updated, _ := model.handleBrowseKey(keyMsg("l"))
	detail := updated.(Model)
	if detail.mode != DetailMode {
		t.Fatalf("mode after l = %v, want DetailMode", detail.mode)
	}

	updated, _ = detail.handleDetailKey(keyMsg("h"))
	browse := updated.(Model)
	if browse.mode != BrowseMode {
		t.Fatalf("mode after h = %v, want BrowseMode", browse.mode)
	}
}

func TestColumnSettingsApplyReorderAndVisibilityChanges(t *testing.T) {
	model := NewModel(
		t.Context(),
		nil,
		[]logrecord.LogRecord{testLogRecord()},
		ModelOptions{AttributeNames: []string{"@http.status_code"}},
	)
	model.width = 80
	model.height = 24

	updated, _ := model.handleBrowseKey(keyMsg("c"))
	settings := updated.(Model)
	if settings.mode != ColumnsMode {
		t.Fatalf("mode = %v, want ColumnsMode", settings.mode)
	}

	updated, _ = settings.handleColumnsKey(keyMsg(" "))
	settings = updated.(Model)
	if settings.columnDraft[0].Visible {
		t.Fatal("timestamp should be hidden in draft")
	}

	updated, _ = settings.handleColumnsKey(keyMsg("J"))
	settings = updated.(Model)
	if settings.columnDraft[1].Name != TimestampColumn {
		t.Fatalf("reordered columns = %+v", settings.columnDraft[:2])
	}

	updated, _ = settings.handleColumnsKey(keyMsg("enter"))
	applied := updated.(Model)
	if applied.mode != BrowseMode {
		t.Fatalf("mode = %v, want BrowseMode", applied.mode)
	}
	if applied.columns[1] != (Column{Name: TimestampColumn, Visible: false}) {
		t.Fatalf("applied columns = %+v", applied.columns[:2])
	}
}

func TestColumnSettingsCancelDiscardsChanges(t *testing.T) {
	model := NewModel(t.Context(), nil, []logrecord.LogRecord{testLogRecord()})
	updated, _ := model.handleBrowseKey(keyMsg("c"))
	settings := updated.(Model)
	updated, _ = settings.handleColumnsKey(keyMsg(" "))
	settings = updated.(Model)
	updated, _ = settings.handleColumnsKey(keyMsg("esc"))
	cancelled := updated.(Model)

	if !cancelled.columns[0].Visible {
		t.Fatal("cancel should preserve timestamp visibility")
	}
}

func TestColumnSettingsKeepsAtLeastOneColumnVisible(t *testing.T) {
	model := NewModel(
		t.Context(),
		nil,
		[]logrecord.LogRecord{testLogRecord()},
		ModelOptions{SelectedColumns: []string{MessageColumn}},
	)
	updated, _ := model.handleBrowseKey(keyMsg("c"))
	settings := updated.(Model)
	updated, _ = settings.handleColumnsKey(keyMsg(" "))
	settings = updated.(Model)

	if !settings.columnDraft[0].Visible {
		t.Fatal("the last visible column must not be hidden")
	}
}

func TestLiveSearchPreservesSelectionUnlessItWasAtEnd(t *testing.T) {
	records := []logrecord.LogRecord{{ID: "1"}, {ID: "2"}}
	model := NewModel(t.Context(), nil, records)
	model.selectedIndex = 0
	model.applySearchResult(SearchResult{Records: append(records, logrecord.LogRecord{ID: "3"}), Total: 3})
	if model.selectedIndex != 0 {
		t.Fatalf("selected index = %d, want 0", model.selectedIndex)
	}

	model.selectedIndex = len(model.records) - 1
	model.applySearchResult(SearchResult{
		Records: append(model.records, logrecord.LogRecord{ID: "4"}), Total: 4,
	})
	if model.selectedIndex != 3 {
		t.Fatalf("selected index = %d, want new end 3", model.selectedIndex)
	}
}

func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

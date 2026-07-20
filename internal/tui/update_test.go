package tui

import (
	"testing"
	"wanwanlog/internal/logrecord"

	tea "github.com/charmbracelet/bubbletea"
)

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
	lineCount := len(formatDetail(testLogRecord(), contentWidth))
	if maxOffset != max(0, lineCount-contentHeight) {
		t.Fatalf(
			"maximum offset = %d, want line count %d - viewport height %d",
			maxOffset,
			lineCount,
			contentHeight,
		)
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

func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

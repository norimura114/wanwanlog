package tui

import (
	"strings"
	"testing"
	"time"
	"wanwanlog/internal/logrecord"

	"github.com/charmbracelet/lipgloss"
)

func TestViewDoesNotOverflowTerminalWidth(t *testing.T) {
	record := testLogRecord()
	model := NewModel(t.Context(), nil, []logrecord.LogRecord{record})
	model.width = 80
	model.height = 24

	for lineNumber, line := range strings.Split(model.View(), "\n") {
		if width := lipgloss.Width(line); width > model.width {
			t.Fatalf("line %d width = %d, want <= %d: %q", lineNumber+1, width, model.width, line)
		}
	}
}

func TestPaneStylesHighlightThePaneControlledByCurrentMode(t *testing.T) {
	tests := []struct {
		name       string
		mode       Mode
		wantList   lipgloss.TerminalColor
		wantDetail lipgloss.TerminalColor
	}{
		{
			name:       "browse highlights list",
			mode:       BrowseMode,
			wantList:   activePaneBorderColor,
			wantDetail: inactivePaneBorderColor,
		},
		{
			name:       "detail highlights detail",
			mode:       DetailMode,
			wantList:   inactivePaneBorderColor,
			wantDetail: activePaneBorderColor,
		},
		{
			name:       "filter leaves both panes inactive",
			mode:       FilterMode,
			wantList:   inactivePaneBorderColor,
			wantDetail: inactivePaneBorderColor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listStyle, detailStyle := paneStyles(tt.mode)

			if got := listStyle.GetBorderTopForeground(); got != tt.wantList {
				t.Errorf("list border color = %v, want %v", got, tt.wantList)
			}
			if got := detailStyle.GetBorderTopForeground(); got != tt.wantDetail {
				t.Errorf("detail border color = %v, want %v", got, tt.wantDetail)
			}
		})
	}
}

func TestBrowseListRendersHeaderAndAttributeColumns(t *testing.T) {
	model := NewModel(
		t.Context(),
		nil,
		[]logrecord.LogRecord{testLogRecord()},
		ModelOptions{
			AttributeNames:  []string{"@http", "@http.status_code", "@missing"},
			SelectedColumns: []string{"status", "@http", "@http.status_code", "@missing"},
		},
	)

	list := model.renderList(80, 10)
	lines := strings.Split(list, "\n")
	if !strings.Contains(lines[0], "status") || !strings.Contains(lines[0], "@http.status_code") {
		t.Fatalf("header = %q", lines[0])
	}
	if !strings.Contains(lines[1], "{object}") || !strings.Contains(lines[1], "204") {
		t.Fatalf("record row = %q", lines[1])
	}
}

func TestColumnSettingsViewDoesNotOverflowTerminalWidth(t *testing.T) {
	model := NewModel(
		t.Context(),
		nil,
		[]logrecord.LogRecord{testLogRecord()},
		ModelOptions{
			AttributeNames: []string{
				"@this.is.an.extremely.long.attribute.name.that.must.be.truncated",
			},
		},
	)
	model.width = 40
	model.height = 12
	updated, _ := model.handleBrowseKey(keyMsg("c"))
	model = updated.(Model)

	for lineNumber, line := range strings.Split(model.View(), "\n") {
		if width := lipgloss.Width(line); width > model.width {
			t.Fatalf("line %d width = %d, want <= %d: %q", lineNumber+1, width, model.width, line)
		}
	}
}

func TestFormatDetailUsesReadableJSON(t *testing.T) {
	lines := formatDetail(testLogRecord(), 40)
	detail := strings.Join(lines, "\n")

	if strings.Contains(detail, "map[") {
		t.Fatalf("detail should not render Go map syntax: %s", detail)
	}
	if !strings.Contains(detail, `"request_id": "req_000004"`) {
		t.Fatalf("detail should pretty-print attributes JSON: %s", detail)
	}
	if !strings.Contains(detail, `"timestamp": "2026-05-10T12:00:03Z"`) {
		t.Fatalf("detail should pretty-print raw JSON: %s", detail)
	}

	for lineNumber, line := range lines {
		if width := lipgloss.Width(line); width > 40 {
			t.Fatalf("detail line %d width = %d, want <= 40: %q", lineNumber+1, width, line)
		}
	}
}

func testLogRecord() logrecord.LogRecord {
	raw := `{"timestamp":"2026-05-10T12:00:03Z","service":"auth","status":"info","message":"auth handled request with status 204","attributes":{"host":{"az":"ap-northeast-1d","name":"app-04"},"http":{"duration_ms":1706,"method":"DELETE","path":"/api/users","status_code":204},"request_id":"req_000004","span_id":"span_043721","trace_id":"trace_051957","user":{"id":39,"name":"dave"}}}`

	return logrecord.LogRecord{
		Timestamp: time.Date(2026, 5, 10, 12, 0, 3, 0, time.UTC),
		Service:   "auth",
		Status:    "info",
		Message:   "auth handled request with status 204",
		Attributes: map[string]any{
			"host": map[string]any{
				"az":   "ap-northeast-1d",
				"name": "app-04",
			},
			"http": map[string]any{
				"duration_ms": 1706,
				"method":      "DELETE",
				"path":        "/api/users",
				"status_code": 204,
			},
			"request_id": "req_000004",
			"span_id":    "span_043721",
			"trace_id":   "trace_051957",
			"user": map[string]any{
				"id":   39,
				"name": "dave",
			},
		},
		Raw: raw,
	}
}

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

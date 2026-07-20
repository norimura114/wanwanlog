package tui

import (
	"strings"
	"testing"
	"wanwanlog/internal/logrecord"
)

func TestAttributeNameCollectorExcludesContainersAndIncludesNestedScalars(t *testing.T) {
	collector := NewAttributeNameCollector()
	collector.Add(map[string]any{
		"http": map[string]any{
			"status": 204,
		},
		"tags": []any{"api", "prod"},
	})

	names := collector.Names()
	if len(names) != 1 || names[0] != "@http.status" {
		t.Fatalf("attribute names = %v, want [@http.status]", names)
	}
}

func TestAttributeNameCollectorIncludesKeySeenAsScalarInAnotherRecord(t *testing.T) {
	collector := NewAttributeNameCollector()
	collector.Add(map[string]any{"value": map[string]any{"nested": true}})
	collector.Add(map[string]any{"value": "scalar"})

	names := collector.Names()
	if !contains(names, "@value") {
		t.Fatalf("attribute names should include key also seen as scalar: %v", names)
	}
}

func TestFormatAttributeColumnValues(t *testing.T) {
	attributes := map[string]any{
		"http": map[string]any{"status": float64(204)},
		"tags": []any{"api"},
	}

	tests := []struct {
		column string
		want   string
	}{
		{column: "@http", want: "{object}"},
		{column: "@http.status", want: "204"},
		{column: "@tags", want: "[array]"},
		{column: "@missing", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.column, func(t *testing.T) {
			got := formatColumnValue(tt.column, attributes, testLogRecord().Timestamp, "", "", "")
			if got != tt.want {
				t.Fatalf("value = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewColumnsUsesCLIOrderAndKeepsOtherChoicesHidden(t *testing.T) {
	columns := NewColumns(
		[]string{"@user.id", "@http.status"},
		[]string{"@http.status", "message"},
	)

	if columns[0] != (Column{Name: "@http.status", Visible: true}) {
		t.Fatalf("first column = %+v", columns[0])
	}
	if columns[1] != (Column{Name: "message", Visible: true}) {
		t.Fatalf("second column = %+v", columns[1])
	}

	for _, column := range columns[2:] {
		if column.Visible {
			t.Fatalf("unselected column should be hidden: %+v", column)
		}
	}
}

func TestBrowseWidthsShrinkMessageFirst(t *testing.T) {
	record := testLogRecord()
	columns := []Column{
		{Name: StatusColumn, Visible: true},
		{Name: MessageColumn, Visible: true},
	}

	unconstrained := browseColumnWidths(columns, []logrecord.LogRecord{record}, 100)
	constrained := browseColumnWidths(columns, []logrecord.LogRecord{record}, 20)

	if constrained[0] != unconstrained[0] {
		t.Fatalf("status width changed from %d to %d before message was exhausted", unconstrained[0], constrained[0])
	}
	if constrained[1] >= unconstrained[1] {
		t.Fatalf("message width = %d, want less than %d", constrained[1], unconstrained[1])
	}

	row := formatBrowseRow(columns, constrained, &record)
	if !strings.Contains(row, "…") {
		t.Fatalf("constrained row should truncate message: %q", row)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

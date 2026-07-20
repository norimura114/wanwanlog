package output

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
	"wanwanlog/internal/logrecord"
)

type TableFormatter struct {
	MaxMessageWidth    int
	MaxAttributesWidth int
	Columns            []string
}

func NewTableFormatter(columns ...string) *TableFormatter {
	return &TableFormatter{
		MaxMessageWidth:    40,
		MaxAttributesWidth: 40,
		Columns:            append([]string(nil), columns...),
	}
}

func (f *TableFormatter) Write(ctx context.Context, w io.Writer, out SearchOutput) error {
	if f.MaxMessageWidth <= 0 {
		f.MaxMessageWidth = 40
	}

	if f.MaxAttributesWidth <= 0 {
		f.MaxAttributesWidth = 40
	}

	if out.Query != "" {
		if _, err := fmt.Fprintf(w, "QUERY %s\n", out.Query); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "TOTAL %d\n\n", out.Total); err != nil {
		return err
	}

	if len(out.Results) == 0 {
		_, err := fmt.Fprintln(w, "No results")
		return err
	}

	if len(f.Columns) > 0 {
		return f.writeSelectedColumns(ctx, w, out.Results)
	}

	if _, err := fmt.Fprintf(
		w,
		"%-19s  %-5s  %-12s  %-40s  %s\n",
		"TIME",
		"LEVEL",
		"SERVICE",
		"MESSAGE",
		"ATTRIBUTES",
	); err != nil {
		return err
	}

	for _, record := range out.Results {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := fmt.Sprintf(
			"%-19s  %-5s  %-12s  %-40s  %s\n",
			formatTime(record.Timestamp),
			formatStatus(record.Status),
			truncate(record.Service, 12),
			truncateOneLine(record.Message, f.MaxMessageWidth),
			formatAttributes(record.Attributes, f.MaxAttributesWidth),
		)

		if _, err := fmt.Fprint(w, line); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintf(w, "\nShowing %d results\n", len(out.Results))
	return err
}

func (f *TableFormatter) writeSelectedColumns(
	ctx context.Context,
	w io.Writer,
	records []logrecord.LogRecord,
) error {
	widths := make([]int, len(f.Columns))
	for i, column := range f.Columns {
		widths[i] = len([]rune(column))
		for _, record := range records {
			width := len([]rune(f.columnValue(column, record)))
			if width > widths[i] {
				widths[i] = width
			}
		}

		maxWidth := f.MaxAttributesWidth
		if column == "message" {
			maxWidth = f.MaxMessageWidth
		}
		if widths[i] > maxWidth {
			widths[i] = maxWidth
		}
	}

	headers := make([]string, len(f.Columns))
	for i, column := range f.Columns {
		headers[i] = strings.ToUpper(column)
	}
	if _, err := fmt.Fprintln(w, formatSelectedRow(headers, widths)); err != nil {
		return err
	}

	for _, record := range records {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		values := make([]string, len(f.Columns))
		for i, column := range f.Columns {
			values[i] = f.columnValue(column, record)
		}
		if _, err := fmt.Fprintln(w, formatSelectedRow(values, widths)); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintf(w, "\nShowing %d results\n", len(records))
	return err
}

func (f *TableFormatter) columnValue(column string, record logrecord.LogRecord) string {
	switch column {
	case "timestamp":
		return formatTime(record.Timestamp)
	case "status":
		return record.Status
	case "service":
		return record.Service
	case "message":
		return truncateOneLine(record.Message, f.MaxMessageWidth)
	}

	value, ok := nestedAttribute(record.Attributes, strings.TrimPrefix(column, "@"))
	if !ok {
		return ""
	}
	return truncateOneLine(formatAttributeValue(value), f.MaxAttributesWidth)
}

func nestedAttribute(attributes map[string]any, path string) (any, bool) {
	var current any = attributes
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func formatSelectedRow(values []string, widths []int) string {
	cells := make([]string, len(values))
	for i, value := range values {
		value = truncateOneLine(value, widths[i])
		cells[i] = fmt.Sprintf("%-*s", widths[i], value)
	}
	return strings.TrimRight(strings.Join(cells, "  "), " ")
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	return t.Local().Format("2006-01-02 15:04:05")
}

func formatStatus(status string) string {
	if status == "" {
		return "-"
	}

	return strings.ToUpper(status)
}

func truncateOneLine(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return truncate(s, max)
}

func truncate(s string, max int) string {
	if max <= 0 {
		return s
	}

	runes := []rune(s)
	if len(runes) <= max {
		return s
	}

	if max <= 1 {
		return "…"
	}

	return string(runes[:max-1]) + "…"
}

func formatAttributes(attributes map[string]any, maxWidth int) string {
	if len(attributes) == 0 {
		return "-"
	}

	flattened := map[string]any{}
	flattenAttributes("", attributes, flattened)

	keys := make([]string, 0, len(flattened))
	for key := range flattened {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, formatAttributeValue(flattened[key])))
	}

	return truncateOneLine(strings.Join(parts, " "), maxWidth)
}

func flattenAttributes(prefix string, value any, out map[string]any) {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			nextKey := key
			if prefix != "" {
				nextKey = prefix + "." + key
			}

			flattenAttributes(nextKey, child, out)
		}
	default:
		out[prefix] = v
	}
}

func formatAttributeValue(value any) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return formatFloat(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(b)
	}
}

func formatFloat(v float64) string {
	s := fmt.Sprintf("%f", v)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")

	if s == "" {
		return "0"
	}

	return s
}

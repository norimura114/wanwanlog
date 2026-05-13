package output

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type TableFormatter struct {
	MaxMessageWidth    int
	MaxAttributesWidth int
}

func NewTableFormatter() *TableFormatter {
	return &TableFormatter{
		MaxMessageWidth:    60,
		MaxAttributesWidth: 40,
	}
}

func (f *TableFormatter) Write(ctx context.Context, w io.Writer, out SearchOutput) error {
	if f.MaxMessageWidth <= 0 {
		f.MaxMessageWidth = 60
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

	if _, err := fmt.Fprintf(
		w,
		"%-19s  %-5s  %-12s  %-60s  %s\n",
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
			"%-19s  %-5s  %-12s  %-60s  %s\n",
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

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	return t.Local().Format("2006-01-012 15:04:05")
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

package output

import (
	"fmt"
	"io"
	"strings"
	"time"
	"wanwanlog/internal/logrecord"
)

func WriteTable(w io.Writer, records []logrecord.LogRecord) error {
	for _, record := range records {
		line := formatLine(record)

		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	return nil
}

func formatLine(record logrecord.LogRecord) string {
	timestamp := formatTimestamp(record.Timestamp)
	status := padRight(record.Status, 5)
	service := padRight(record.Service, 12)
	message := strings.ReplaceAll(record.Message, "\n", " ")

	return fmt.Sprintf("%s %s %s %s", timestamp, status, service, message)
}

func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	return t.Format(time.RFC3339)
}

func padRight(value string, width int) string {
	if len(value) >= width {
		return value
	}

	return value + strings.Repeat(" ", width-len(value))
}

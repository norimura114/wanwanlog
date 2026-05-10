package logrecord

import "time"

type LogRecord struct {
	Timestamp time.Time
	Message   string
	Level     string
	Service   string
	Attrs     map[string]any
	Raw       map[string]any
}

func New(
	timestamp time.Time,
	message string,
	level string,
	service string,
	attrs map[string]any,
	raw map[string]any,
) LogRecord {
	if attrs == nil {
		attrs = map[string]any{}
	}

	if raw == nil {
		raw = map[string]any{}
	}

	return LogRecord{
		Timestamp: timestamp,
		Message:   message,
		Level:     level,
		Service:   service,
		Attrs:     attrs,
		Raw:       raw,
	}
}

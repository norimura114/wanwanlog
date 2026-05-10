package logrecord

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	ts := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	record := New(
		ts,
		"request completed",
		"info",
		"api",
		map[string]any{
			"http.status_code": float64(200),
		},
		map[string]any{
			"message": "request completed",
			"level":   "info",
		},
	)

	if !record.Timestamp.Equal(ts) {
		t.Fatalf("expected timestamp %v, got %v", ts, record.Timestamp)
	}

	if record.Message != "request completed" {
		t.Fatalf("expected message %q, got %q", "request completed", record.Message)
	}

	if record.Level != "info" {
		t.Fatalf("expected level %q, got %q", "info", record.Level)
	}

	if record.Service != "api" {
		t.Fatalf("expected service %q, got %q", "api", record.Service)
	}

	if record.Attrs["http.status_code"] != float64(200) {
		t.Fatalf("expected status code 200, got %v", record.Attrs["http.status_code"])
	}

	if record.Raw["message"] != "request completed" {
		t.Fatalf("expected raw message, got %v", record.Raw["message"])
	}
}

func TestNewWithNilMaps(t *testing.T) {
	record := New(
		time.Time{},
		"",
		"",
		"",
		nil,
		nil,
	)

	if record.Attrs == nil {
		t.Fatal("expected Attrs to be initialized")
	}

	if record.Raw == nil {
		t.Fatal("expected Raw to be initialized")
	}
}

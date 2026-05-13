package search

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"wanwanlog/internal/logrecord"
)

func TestSQLiteStore_InsertBatch(t *testing.T) {
	ctx := context.Background()

	store, err := OpenSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	records := []logrecord.LogRecord{
		{
			ID:        "log_001",
			Timestamp: time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
			Service:   "api",
			Message:   "failed to connect database",
			Attributes: map[string]any{
				"user": map[string]any{
					"id": "123",
				},
				"http": map[string]any{
					"method":      "GET",
					"status_code": 500,
				},
			},
			Raw: `{"timestamp":"2026-05-10T12:00:00Z","service":"api","status":"error","message":"failed to connect database"}`,
		},
		{
			ID:        "log_002",
			Timestamp: time.Date(2026, 5, 10, 12, 1, 0, 0, time.UTC),
			Service:   "worker",
			Status:    "info",
			Message:   "job completed",
			Attributes: map[string]any{
				"job_id": "job_001",
			},
			Raw: `{"timestamp":"2026-05-10T12:01:00Z","service":"worker","status":"info","message":"job completed"}`,
		},
	}

	if err := store.InsertBatch(ctx, records); err != nil {
		t.Fatalf("failed to insert batch: %v", err)
	}

	var count int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM logs;`).Scan(&count); err != nil {
		t.Fatalf("failed to count logs: %v", err)
	}

	if count != 2 {
		t.Errorf("count %d, want %d", count, 2)
	}
}

func TestSQLiteStore_InsertBatch_InsertsLogFields(t *testing.T) {
	ctx := context.Background()

	store, err := OpenSQLiteStore(":memory")
	if err != nil {
		t.Fatalf("failed to open sqlite store: %v", err)
	}
	defer store.Close()

	record := logrecord.LogRecord{
		ID:        "log_001",
		Timestamp: time.Date(2026, 5, 10, 12, 0, 0, 123456789, time.UTC),
		Service:   "api",
		Status:    "error",
		Message:   "failed to connect database",
		Attributes: map[string]any{
			"user_id": "123",
		},
		Raw: `{"message":"failed to connect database"}`,
	}

	if err := store.InsertBatch(ctx, []logrecord.LogRecord{record}); err != nil {
		t.Fatalf("failed to insert batch: %v", err)
	}

	row := store.db.QueryRowContext(ctx, `
SELECT id, timestamp, service, status, message, attributes, raw
FROM logs
WHERE id = ?;
`, "log_001")

	var id string
	var timestamp string
	var service string
	var status string
	var message string
	var attributesJSON string
	var raw string

	if err := row.Scan(
		&id,
		&timestamp,
		&service,
		&status,
		&message,
		&attributesJSON,
		&raw,
	); err != nil {
		t.Fatalf("failed to scan log: %v", err)
	}

	if id != "log_001" {
		t.Errorf("id = %q, want %q", id, "log_001")
	}

	if timestamp != "2026-05-10T12:00:00.123456789Z" {
		t.Errorf("timestamp = %q, want %q", timestamp, "2026-05-10T12:00:00.123456789Z")
	}

	if service != "api" {
		t.Errorf("service = %q, want %q", service, "api")
	}

	if status != "error" {
		t.Errorf("status = %q, want %q", status, "error")
	}

	if message != "failed to connect database" {
		t.Errorf("message = %q, want %q", message, "failed to connect database")
	}

	var attributes map[string]any
	if err := json.Unmarshal([]byte(attributesJSON), &attributes); err != nil {
		t.Fatalf("failed to unmarshal attributes: %v", err)
	}

	if attributes["user_id"] != "123" {
		t.Errorf("attributes[user_id] = %v, want %v", attributes["user_id"], "123")
	}

	if raw != `{"message":"failed to connect database"}` {
		t.Errorf("raw = %q, want %q", raw, `{"message":"failed to connect database"}`)
	}
}

func TestSQLiteStore_InsertBatch_EmptyRecords(t *testing.T) {
	ctx := context.Background()

	store, err := OpenSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite store: %v", err)
	}
	defer store.Close()

	if err := store.InsertBatch(ctx, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

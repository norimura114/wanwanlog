package search

import (
	"context"
	"testing"
	"time"
	"wanwanlog/internal/filter"
	"wanwanlog/internal/logrecord"
)

func TestSQLiteStoreSearchByField(t *testing.T) {
	ctx := context.Background()

	store, err := OpenSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := store.Init(ctx); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	records := []logrecord.LogRecord{
		{
			ID:        "1",
			Timestamp: mustParseTime(t, "2026-05-10T12:00:00Z"),
			Service:   "api",
			Status:    "error",
			Message:   "failed to connect database",
			Attributes: map[string]any{
				"user": map[string]any{
					"id": "123",
				},
			},
			Raw: `{"service":"api","status":"error","message":"failed to connect database"}`,
		},
		{
			ID:        "2",
			Timestamp: mustParseTime(t, "2026-05-10T12:01:00Z"),
			Service:   "worker",
			Status:    "info",
			Message:   "job completed",
			Attributes: map[string]any{
				"user": map[string]any{
					"id": "456",
				},
			},
			Raw: `{"service":"worker","status":"info","message":"job completed"}`,
		},
	}

	if err := store.InsertBatch(ctx, records); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	result, err := store.Search(ctx, filter.SearchQuery{
		Filter: "service:api",
		Limit:  100,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}

	if len(result.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result.Records))
	}

	if result.Records[0].ID != "1" {
		t.Fatalf("expected record ID 1, got %s", result.Records[0].ID)
	}
}

func TestSQLiteStoreSearchByAttribute(t *testing.T) {
	ctx := context.Background()

	store, err := OpenSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := store.Init(ctx); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	records := []logrecord.LogRecord{
		{
			ID:        "1",
			Timestamp: mustParseTime(t, "2026-05-10T12:00:00Z"),
			Service:   "api",
			Status:    "error",
			Message:   "failed to connect database",
			Attributes: map[string]any{
				"user": map[string]any{
					"id": "123",
				},
			},
			Raw: `{"service":"api","status":"error","message":"failed to connect database"}`,
		},
		{
			ID:        "2",
			Timestamp: mustParseTime(t, "2026-05-10T12:01:00Z"),
			Service:   "api",
			Status:    "error",
			Message:   "failed to connect cache",
			Attributes: map[string]any{
				"user": map[string]any{
					"id": "456",
				},
			},
			Raw: `{"service":"api","status":"error","message":"failed to connect cache"}`,
		},
	}

	if err := store.InsertBatch(ctx, records); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	result, err := store.Search(ctx, filter.SearchQuery{
		Filter: "@user.id:123",
		Limit:  100,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}

	if result.Records[0].ID != "1" {
		t.Fatalf("expected record ID 1, got %s", result.Records[0].ID)
	}
}

func TestSQLiteStoreSearchByKeyword(t *testing.T) {
	ctx := context.Background()

	store, err := OpenSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := store.Init(ctx); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	records := []logrecord.LogRecord{
		{
			ID:         "1",
			Timestamp:  mustParseTime(t, "2026-05-10T12:00:00Z"),
			Service:    "api",
			Status:     "error",
			Message:    "failed to connect database",
			Attributes: map[string]any{},
			Raw:        `{"message":"failed to connect database"}`,
		},
		{
			ID:         "2",
			Timestamp:  mustParseTime(t, "2026-05-10T12:01:00Z"),
			Service:    "api",
			Status:     "info",
			Message:    "request completed",
			Attributes: map[string]any{},
			Raw:        `{"message":"request completed"}`,
		},
	}

	if err := store.InsertBatch(ctx, records); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	result, err := store.Search(ctx, filter.SearchQuery{
		Filter: "database",
		Limit:  100,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}

	if result.Records[0].ID != "1" {
		t.Fatalf("expected record ID 1, got %s", result.Records[0].ID)
	}
}

func TestSQLiteStoreSearchByAndAndNot(t *testing.T) {
	ctx := context.Background()

	store, err := OpenSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := store.Init(ctx); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	records := []logrecord.LogRecord{
		{
			ID:         "1",
			Timestamp:  mustParseTime(t, "2026-05-10T12:00:00Z"),
			Service:    "api",
			Status:     "error",
			Message:    "failed to connect database",
			Attributes: map[string]any{},
			Raw:        `{"service":"api","status":"error"}`,
		},
		{
			ID:         "2",
			Timestamp:  mustParseTime(t, "2026-05-10T12:01:00Z"),
			Service:    "api",
			Status:     "info",
			Message:    "request completed",
			Attributes: map[string]any{},
			Raw:        `{"service":"api","status":"info"}`,
		},
		{
			ID:         "3",
			Timestamp:  mustParseTime(t, "2026-05-10T12:02:00Z"),
			Service:    "worker",
			Status:     "error",
			Message:    "job failed",
			Attributes: map[string]any{},
			Raw:        `{"service":"worker","status":"error"}`,
		},
	}

	if err := store.InsertBatch(ctx, records); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	result, err := store.Search(ctx, filter.SearchQuery{
		Filter: "service:api -status:info",
		Limit:  100,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}

	if result.Records[0].ID != "1" {
		t.Fatalf("expected record ID 1, got %s", result.Records[0].ID)
	}
}

func TestSQLiteStoreSearchEmptyFilter(t *testing.T) {
	ctx := context.Background()

	store, err := OpenSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := store.Init(ctx); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	records := []logrecord.LogRecord{
		{
			ID:         "1",
			Timestamp:  mustParseTime(t, "2026-05-10T12:00:00Z"),
			Service:    "api",
			Status:     "error",
			Message:    "failed",
			Attributes: map[string]any{},
			Raw:        `{"message":"failed"}`,
		},
		{
			ID:         "2",
			Timestamp:  mustParseTime(t, "2026-05-10T12:01:00Z"),
			Service:    "api",
			Status:     "info",
			Message:    "completed",
			Attributes: map[string]any{},
			Raw:        `{"message":"completed"}`,
		},
	}

	if err := store.InsertBatch(ctx, records); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	result, err := store.Search(ctx, filter.SearchQuery{
		Filter: "",
		Limit:  100,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}

	if len(result.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(result.Records))
	}
}

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatalf("failed to parse time: %v", err)
	}

	return parsed
}

package search

import (
	"context"
	"fmt"
	"testing"
	"time"
	"wanwanlog/internal/filter"
	"wanwanlog/internal/logrecord"
)

func TestSQLiteStoreLatestWindowIsDisplayedOldestFirst(t *testing.T) {
	store, err := OpenSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	records := make([]logrecord.LogRecord, 105)
	for i := range records {
		records[i] = logrecord.LogRecord{
			ID: fmt.Sprint(i), Timestamp: time.Unix(int64(i), 0),
			Attributes: map[string]any{}, Raw: "{}",
		}
	}
	if err := store.InsertBatch(context.Background(), records); err != nil {
		t.Fatal(err)
	}
	result, err := store.Search(context.Background(), filter.SearchQuery{
		Limit: 100, Order: filter.LatestWindow,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 105 || len(result.Records) != 100 {
		t.Fatalf("total=%d records=%d", result.Total, len(result.Records))
	}
	if result.Records[0].ID != "5" || result.Records[99].ID != "104" {
		t.Fatalf("window starts=%s ends=%s", result.Records[0].ID, result.Records[99].ID)
	}
}

package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
	"wanwanlog/internal/app/usecase"
	"wanwanlog/internal/filter"
	"wanwanlog/internal/parse"
	"wanwanlog/internal/search"
	"wanwanlog/internal/source"
	"wanwanlog/internal/tui"
)

func TestLiveFollowIngestionAcrossAppendTruncateAndRename(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.jsonl")
	writeFile(t, path, `{"id":"initial","message":"initial padding padding"}`+"\n")

	store, err := search.OpenSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	loader := usecase.NewLoadLogs(parse.NewJSONLinesParser(), store, 1000)
	application := &TUIApplication{Loader: loader, Store: store}
	follower := source.NewLocalFileFollower(path, 5*time.Millisecond)
	initial, err := follower.InitialRead(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loader.Load(t.Context(), bytes.NewReader(initial.Lines)); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	events := make(chan tui.FollowEvent, 16)
	ingestDone := make(chan struct{})
	go follower.Run(ctx)
	go func() {
		defer close(ingestDone)
		application.ingestEvents(ctx, follower.Events(), events)
	}()

	appendFile(t, path, `{"id":"split"`)
	assertNoDataEvent(t, events, 20*time.Millisecond)
	appendFile(t, path, `,"new":{"scalar":1}}`+"\n")
	event := waitDataEvent(t, events)
	if event.AddedCount != 1 || event.ParseErrorCount != 0 {
		t.Fatalf("split append event = %+v", event)
	}

	appendFile(t, path, "not-json\n"+`{"id":"normal"}`+"\n")
	event = waitDataEvent(t, events)
	if event.AddedCount != 1 || event.ParseErrorCount != 1 {
		t.Fatalf("mixed append event = %+v", event)
	}

	writeFile(t, path, `{"id":"truncated"}`+"\n")
	event = waitResetEvent(t, events)
	if event.AddedCount != 1 {
		t.Fatalf("truncate event = %+v", event)
	}
	assertStoreIDs(t, store, "truncated")

	if err := os.Rename(path, path+".1"); err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, `{"id":"replacement"}`+"\n")
	event = waitResetEvent(t, events)
	if event.AddedCount != 1 {
		t.Fatalf("rename event = %+v", event)
	}
	assertStoreIDs(t, store, "replacement")

	cancel()
	if err := follower.Close(); err != nil {
		t.Fatal(err)
	}
	<-ingestDone
}

func assertStoreIDs(t *testing.T, store search.Store, want ...string) {
	t.Helper()
	result, err := store.Search(t.Context(), filter.SearchQuery{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Records) != len(want) {
		t.Fatalf("records = %v, want IDs %v", result.Records, want)
	}
	for i := range want {
		if result.Records[i].ID != want[i] {
			t.Fatalf("record %d ID = %s, want %s", i, result.Records[i].ID, want[i])
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func appendFile(t *testing.T, path, content string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func waitDataEvent(t *testing.T, events <-chan tui.FollowEvent) tui.FollowEvent {
	t.Helper()
	timeout := time.After(time.Second)
	for {
		select {
		case event := <-events:
			if event.AddedCount > 0 || event.ParseErrorCount > 0 {
				return event
			}
		case <-timeout:
			t.Fatal("timed out waiting for data event")
		}
	}
}

func waitResetEvent(t *testing.T, events <-chan tui.FollowEvent) tui.FollowEvent {
	t.Helper()
	timeout := time.After(time.Second)
	for {
		select {
		case event := <-events:
			if event.Reset {
				return event
			}
		case <-timeout:
			t.Fatal("timed out waiting for reset event")
		}
	}
}

func assertNoDataEvent(t *testing.T, events <-chan tui.FollowEvent, duration time.Duration) {
	t.Helper()
	timeout := time.After(duration)
	for {
		select {
		case event := <-events:
			if event.AddedCount > 0 || event.ParseErrorCount > 0 {
				t.Fatalf("unexpected data event: %+v", event)
			}
		case <-timeout:
			return
		}
	}
}

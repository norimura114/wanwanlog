package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLocalFileFollowerWaitsForCompleteLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs.jsonl")
	if err := os.WriteFile(path, []byte("{\"id\":\"initial\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	follower := NewLocalFileFollower(path, 5*time.Millisecond)
	go follower.Run(ctx)
	waitForLive(t, follower.Events())

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(`{"id":"next"`); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if _, err := file.WriteString("}\n"); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()

	event := waitForLines(t, follower.Events())
	if string(event.Lines) != "{\"id\":\"next\"}\n" {
		t.Fatalf("lines = %q", event.Lines)
	}
}

func TestLocalFileFollowerInitialReadRetainsTrailingFragment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs.jsonl")
	if err := os.WriteFile(path, []byte("{\"id\":\"one\"}\n{\"id\":\"two\""), 0o600); err != nil {
		t.Fatal(err)
	}
	follower := NewLocalFileFollower(path, 5*time.Millisecond)
	initial, err := follower.InitialRead(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if string(initial.Lines) != "{\"id\":\"one\"}\n" {
		t.Fatalf("initial lines = %q", initial.Lines)
	}

	ctx, cancel := context.WithCancel(t.Context())
	go follower.Run(ctx)
	appendFileForFollower(t, path, "}\n")
	event := waitForLines(t, follower.Events())
	if string(event.Lines) != "{\"id\":\"two\"}\n" {
		t.Fatalf("completed lines = %q", event.Lines)
	}
	cancel()
	if err := follower.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLocalFileFollowerResetsAfterRename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "logs.jsonl")
	if err := os.WriteFile(path, []byte("{\"id\":\"old\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	follower := NewLocalFileFollower(path, 5*time.Millisecond)
	go follower.Run(ctx)
	waitForLive(t, follower.Events())

	if err := os.Rename(path, path+".1"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{\"id\":\"new\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	event := waitForLines(t, follower.Events())
	if !event.Reset || string(event.Lines) != "{\"id\":\"new\"}\n" {
		t.Fatalf("event = %+v", event)
	}
}

func TestLocalFileFollowerRetriesMissingPathAndReadsReplacement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "logs.jsonl")
	if err := os.WriteFile(path, []byte("{\"id\":\"old\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	follower := NewLocalFileFollower(path, 5*time.Millisecond)
	if _, err := follower.InitialRead(t.Context()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go follower.Run(ctx)

	if err := os.Rename(path, path+".1"); err != nil {
		t.Fatal(err)
	}
	waitForRetry(t, follower.Events())
	if err := os.WriteFile(path, []byte("{\"id\":\"new\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	event := waitForLines(t, follower.Events())
	if !event.Reset || string(event.Lines) != "{\"id\":\"new\"}\n" {
		t.Fatalf("replacement event = %+v", event)
	}
}

func waitForLive(t *testing.T, events <-chan FollowBatch) {
	t.Helper()
	select {
	case event := <-events:
		if event.State != FollowLive {
			t.Fatalf("event = %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for live event")
	}
}

func waitForLines(t *testing.T, events <-chan FollowBatch) FollowBatch {
	t.Helper()
	timeout := time.After(time.Second)
	for {
		select {
		case event := <-events:
			if len(event.Lines) > 0 {
				return event
			}
		case <-timeout:
			t.Fatal("timed out waiting for lines")
		}
	}
}

func waitForRetry(t *testing.T, events <-chan FollowBatch) {
	t.Helper()
	timeout := time.After(time.Second)
	for {
		select {
		case event := <-events:
			if event.State == FollowRetrying {
				return
			}
		case <-timeout:
			t.Fatal("timed out waiting for retry")
		}
	}
}

func appendFileForFollower(t *testing.T, path, content string) {
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

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunFollowAppendsWithoutTruncating(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs.jsonl")
	const existing = `{"message":"existing"}` + "\n"
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := runFollow(t.Context(), path, 2, 1, time.Millisecond); err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var lines [][]byte
	for scanner.Scan() {
		lines = append(lines, append([]byte(nil), scanner.Bytes()...))
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 || string(lines[0]) != `{"message":"existing"}` {
		t.Fatalf("lines = %q", lines)
	}

	var appended LogLine
	if err := json.Unmarshal(lines[1], &appended); err != nil {
		t.Fatal(err)
	}
	if appended.Attributes["request_id"] != "req_000002" {
		t.Fatalf("request_id = %v, want req_000002", appended.Attributes["request_id"])
	}
}

func TestRunFollowCountZeroStopsOnCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs.jsonl")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- runFollow(ctx, path, 0, 1, time.Millisecond)
	}()

	deadline := time.Now().Add(time.Second)
	for {
		info, err := os.Stat(path)
		if err == nil && info.Size() > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("follow mode did not append a line")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("follow mode did not stop after cancellation")
	}
}

func TestRunFollowRejectsIncompleteExistingLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs.jsonl")
	if err := os.WriteFile(path, []byte(`{"partial":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runFollow(t.Context(), path, 1, 1, time.Millisecond); err == nil {
		t.Fatal("expected incomplete existing line to be rejected")
	}
}

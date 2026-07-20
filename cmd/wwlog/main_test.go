package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectAttributeNamesScansEntireInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs.jsonl")
	content := "{\"message\":\"first\",\"http\":{\"status\":200}}\n" +
		"{\"message\":\"second\",\"late\":{\"value\":true},\"tags\":[\"prod\"]}\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	names, err := collectAttributeNames(path)
	if err != nil {
		t.Fatal(err)
	}

	got := make(map[string]bool, len(names))
	for _, name := range names {
		got[name] = true
	}
	for _, want := range []string{
		"@http.status",
		"@late.value",
	} {
		if !got[want] {
			t.Errorf("attribute names do not include %q: %v", want, names)
		}
	}
	for _, unwanted := range []string{"@http", "@late", "@tags"} {
		if got[unwanted] {
			t.Errorf("attribute names should not include container %q: %v", unwanted, names)
		}
	}
}

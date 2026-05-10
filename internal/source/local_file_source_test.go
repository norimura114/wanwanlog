package source

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalFileSource_Open(t *testing.T) {
	t.Run("opens local file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "app.log")

		content := strings.Join([]string{
			`{"message":"hello"}`,
			`{"message":"world"}`,
		}, "\n")

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		src := NewLocalFileSource(path)

		reader, err := src.Open(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer reader.Close()

		got, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		if string(got) != content {
			t.Fatalf("expected %q, got %q", content, string(got))
		}
	})

	t.Run("returns error when path is empty", func(t *testing.T) {
		src := NewLocalFileSource("")

		_, err := src.Open(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when file does not exist", func(t *testing.T) {
		src := NewLocalFileSource("not-found.log")

		_, err := src.Open(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when context is canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		src := NewLocalFileSource("dummy.log")

		_, err := src.Open(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

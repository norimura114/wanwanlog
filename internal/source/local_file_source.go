package source

import (
	"context"
	"fmt"
	"io"
	"os"
)

var _ Source = (*LocalFileSource)(nil)

type LocalFileSource struct {
	Path string
}

func NewLocalFileSource(path string) *LocalFileSource {
	return &LocalFileSource{
		Path: path,
	}
}

func (s *LocalFileSource) Open(ctx context.Context) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if s.Path == "" {
		return nil, fmt.Errorf("local file path is required")
	}

	file, err := os.Open(s.Path)
	if err != nil {
		return nil, fmt.Errorf("open local file %q: %w", s.Path, err)
	}

	return file, nil
}

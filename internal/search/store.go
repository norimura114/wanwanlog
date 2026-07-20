package search

import (
	"context"
	"wanwanlog/internal/filter"
	"wanwanlog/internal/logrecord"
)

type Store interface {
	Init(ctx context.Context) error
	InsertBatch(ctx context.Context, records []logrecord.LogRecord) error
	Search(ctx context.Context, query filter.SearchQuery) (filter.SearchResult, error)
	DeleteAll(ctx context.Context) error
	Close() error
}

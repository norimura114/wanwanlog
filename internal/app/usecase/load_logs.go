package usecase

import (
	"context"
	"fmt"
	"io"
	"wanwanlog/internal/logrecord"
	"wanwanlog/internal/parse"
	"wanwanlog/internal/search"
)

type LoadLogs struct {
	parser    parse.Parser
	store     search.Store
	batchSize int
}

func NewLoadLogs(
	parser parse.Parser,
	store search.Store,
	batchSize int,
) *LoadLogs {
	if batchSize <= 0 {
		batchSize = 1000
	}

	return &LoadLogs{
		parser:    parser,
		store:     store,
		batchSize: batchSize,
	}
}

type LoadLogsResult struct {
	LoadedCount int
	ErrorCount  int
}

func (u *LoadLogs) Execute(
	ctx context.Context,
	r io.Reader,
) (LoadLogsResult, error) {
	result := LoadLogsResult{}
	batch := make([]logrecord.LogRecord, 0, u.batchSize)
	it := u.parser.Parse(r)

	for it.Next() {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		parsed := it.Result()
		if parsed.Err != nil {
			result.ErrorCount++
			continue
		}

		batch = append(batch, parsed.Log.Record)

		if len(batch) >= u.batchSize {
			if err := u.store.InsertBatch(ctx, batch); err != nil {
				return result, fmt.Errorf("insert batch %w", err)
			}

			result.LoadedCount += len(batch)
			batch = batch[:0]
		}
	}

	if err := it.Err(); err != nil {
		return result, err
	}

	if len(batch) > 0 {
		if err := u.store.InsertBatch(ctx, batch); err != nil {
			return result, fmt.Errorf("insert batch: %w", err)
		}

		result.LoadedCount += len(batch)
	}

	return result, nil
}

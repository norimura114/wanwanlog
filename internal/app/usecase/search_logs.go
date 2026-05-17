package usecase

import (
	"context"
	"fmt"
	"io"
	"wanwanlog/internal/filter"
	"wanwanlog/internal/logrecord"
	"wanwanlog/internal/parse"
	"wanwanlog/internal/search"
	"wanwanlog/internal/source"
)

type SearchLogsInput struct {
	InputPath string
	Query     string
	Limit     int
}

type SearchLogsOutput struct {
	Result filter.SearchResult
}

type SearchLogsUseCase struct {
	SourceFactory func(path string) source.Source
	Parser        parse.Parser
	Store         search.Store
	BatchSize     int
}

func NewSearchLogsUseCase(
	sourceFactory func(path string) source.Source,
	parser parse.Parser,
	store search.Store,
) *SearchLogsUseCase {
	return &SearchLogsUseCase{
		SourceFactory: sourceFactory,
		Parser:        parser,
		Store:         store,
		BatchSize:     1000,
	}
}

func (u *SearchLogsUseCase) Execute(ctx context.Context, input SearchLogsInput) (SearchLogsOutput, error) {
	if input.InputPath == "" {
		return SearchLogsOutput{}, fmt.Errorf("input path is required")
	}

	if input.Limit <= 0 {
		input.Limit = 100
	}

	if u.BatchSize <= 0 {
		u.BatchSize = 1000
	}

	if err := u.Store.Init(ctx); err != nil {
		return SearchLogsOutput{}, fmt.Errorf("init store: %w", err)
	}

	source := u.SourceFactory(input.InputPath)

	reader, err := source.Open(ctx)
	if err != nil {
		return SearchLogsOutput{}, fmt.Errorf("open source :%w", err)
	}
	defer reader.Close()

	// decoder

	if err := u.load(ctx, reader); err != nil {
		return SearchLogsOutput{}, err
	}

	result, err := u.Store.Search(ctx, filter.SearchQuery{
		Filter: input.Query,
		Limit:  input.Limit,
	})
	if err != nil {
		return SearchLogsOutput{}, fmt.Errorf("search logs: %w", err)
	}

	return SearchLogsOutput{
		Result: result,
	}, nil
}

func (u *SearchLogsUseCase) load(ctx context.Context, reader io.Reader) error {
	batch := make([]logrecord.LogRecord, 0, u.BatchSize)
	it := u.Parser.Parse(reader)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}

		if err := u.Store.InsertBatch(ctx, batch); err != nil {
			return fmt.Errorf("insert batch: %w", err)
		}

		batch = batch[:0]
		return nil
	}

	for it.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		result := it.Result()
		if result.Err != nil {
			return fmt.Errorf("parse logs: %w", result.Err)
		}

		batch = append(batch, result.Log.Record)

		if len(batch) >= u.BatchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}

	if err := it.Err(); err != nil {
		return fmt.Errorf("parse logs: %w", err)
	}

	if err := flush(); err != nil {
		return err
	}

	return nil
}

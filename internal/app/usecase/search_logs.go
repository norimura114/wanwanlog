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
	recordsCh, errCh := u.Parser.Parse(ctx, reader)

	batch := make([]logrecord.LogRecord, 0, u.BatchSize)

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

	for recordsCh != nil || errCh != nil {
		select {
		case record, ok := <-recordsCh:
			if !ok {
				recordsCh = nil
				continue
			}

			batch = append(batch, record.Record)

			if len(batch) >= u.BatchSize {
				if err := flush(); err != nil {
					return err
				}
			}

		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}

			return fmt.Errorf("parse logs: %w", err)

		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if err := flush(); err != nil {
		return err
	}

	return nil
}

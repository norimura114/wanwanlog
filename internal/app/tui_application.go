package app

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"
	"wanwanlog/internal/app/usecase"
	"wanwanlog/internal/filter"
	"wanwanlog/internal/search"
	"wanwanlog/internal/source"
	"wanwanlog/internal/tui"
)

type TUIConfig struct {
	FilePath string
	Query    string
	Columns  []string
	Follow   bool
}

type TUIApplication struct {
	Loader       *usecase.LoadLogs
	Searcher     *usecase.SearchLogsUseCase
	Store        search.Store
	PollInterval time.Duration
}

func (a *TUIApplication) Run(ctx context.Context, config TUIConfig) error {
	file, err := os.Open(config.FilePath)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("stat input: %w", err)
	}
	loadResult, err := a.Loader.Execute(ctx, file)
	_ = file.Close()
	if err != nil {
		return fmt.Errorf("load logs: %w", err)
	}

	searchFunc := tui.SearchFunc(func(ctx context.Context, query string) (tui.SearchResult, error) {
		result, err := a.Searcher.Execute(ctx, usecase.SearchLogsInput{
			Query: query, Limit: 100, Order: filter.LatestWindow,
		})
		if err != nil {
			return tui.SearchResult{}, err
		}
		return tui.SearchResult{Records: result.Result.Records, Total: result.Result.Total}, nil
	})
	initial, err := searchFunc(ctx, config.Query)
	if err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var events <-chan tui.FollowEvent
	if config.Follow {
		eventChannel := make(chan tui.FollowEvent, 16)
		events = eventChannel
		follower := source.NewLocalFileFollowerAtOffset(config.FilePath, a.PollInterval, info.Size())
		go follower.Run(runCtx)
		go a.ingestEvents(runCtx, follower.Events(), eventChannel)
	}

	return tui.Run(runCtx, searchFunc, initial, tui.ModelOptions{
		AttributeNames: loadResult.AttributeNames, SelectedColumns: config.Columns,
		InitialQuery: config.Query, FollowEvents: events, Live: config.Follow,
	})
}

func (a *TUIApplication) ingestEvents(ctx context.Context, batches <-chan source.FollowBatch, out chan<- tui.FollowEvent) {
	defer close(out)
	for batch := range batches {
		event := tui.FollowEvent{Reset: batch.Reset, State: string(batch.State), Err: batch.Err}
		if batch.Reset {
			if err := a.Store.DeleteAll(ctx); err != nil {
				event.Err = err
			}
		}
		if len(batch.Lines) > 0 {
			lines, oversized := validSizedLines(batch.Lines)
			result, err := a.Loader.Execute(ctx, bytes.NewReader(lines))
			event.AddedCount = result.LoadedCount
			event.ParseErrorCount = result.ErrorCount + oversized
			event.AttributeNames = result.AttributeNames
			if err != nil {
				event.Err = err
			}
		}
		select {
		case out <- event:
		case <-ctx.Done():
			return
		}
	}
}

func validSizedLines(data []byte) ([]byte, int) {
	const maxLineSize = 10 * 1024 * 1024
	var valid bytes.Buffer
	errors := 0
	for len(data) > 0 {
		index := bytes.IndexByte(data, '\n')
		if index < 0 {
			break
		}
		line := data[:index]
		data = data[index+1:]
		if len(line) > maxLineSize {
			errors++
			continue
		}
		valid.Write(line)
		valid.WriteByte('\n')
	}
	return valid.Bytes(), errors
}

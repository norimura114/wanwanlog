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
	var (
		loadResult usecase.LoadResult
		follower   *source.LocalFileFollower
		err        error
	)
	if config.Follow {
		follower = source.NewLocalFileFollower(config.FilePath, a.PollInterval)
		initial, initialErr := follower.InitialRead(ctx)
		if initialErr != nil {
			return initialErr
		}
		lines, oversized := validSizedLines(initial.Lines)
		loadResult, err = a.Loader.Load(ctx, bytes.NewReader(lines))
		loadResult.ErrorCount += initial.OversizedLineCount + oversized
	} else {
		var file *os.File
		file, err = os.Open(config.FilePath)
		if err == nil {
			loadResult, err = a.Loader.Load(ctx, file)
			_ = file.Close()
		}
	}
	if err != nil {
		if follower != nil {
			_ = follower.Discard()
		}
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
		if follower != nil {
			_ = follower.Discard()
		}
		return err
	}

	runCtx, cancel := context.WithCancel(ctx)
	var events <-chan tui.FollowEvent
	var ingestDone chan struct{}
	if config.Follow {
		eventChannel := make(chan tui.FollowEvent, 16)
		events = eventChannel
		ingestDone = make(chan struct{})
		go follower.Run(runCtx)
		go func() {
			defer close(ingestDone)
			a.ingestEvents(runCtx, follower.Events(), eventChannel)
		}()
	}

	runErr := tui.Run(runCtx, searchFunc, initial, tui.ModelOptions{
		AttributeNames: loadResult.AttributeNames, SelectedColumns: config.Columns,
		InitialQuery: config.Query, FollowEvents: events, Live: config.Follow,
		ParseErrorCount: loadResult.ErrorCount,
	})
	cancel()
	if follower != nil {
		_ = follower.Close()
		<-ingestDone
	}
	return runErr
}

func (a *TUIApplication) ingestEvents(ctx context.Context, batches <-chan source.FollowBatch, out chan<- tui.FollowEvent) {
	defer close(out)
	for batch := range batches {
		event := tui.FollowEvent{Reset: batch.Reset, State: string(batch.State), Err: batch.Err}
		if batch.Reset {
			if err := a.Store.DeleteAll(ctx); err != nil {
				event.Err = err
				event.State = "error"
			}
		}
		if len(batch.Lines) > 0 {
			lines, oversized := validSizedLines(batch.Lines)
			result, err := a.Loader.Execute(ctx, bytes.NewReader(lines))
			event.AddedCount = result.LoadedCount
			event.ParseErrorCount = result.ErrorCount + oversized + batch.OversizedLineCount
			event.AttributeNames = result.AttributeNames
			if err != nil {
				event.Err = err
				event.State = "error"
			}
		}
		if len(batch.Lines) == 0 {
			event.ParseErrorCount += batch.OversizedLineCount
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

package main

import (
	"context"
	"fmt"
	"os"
	"wanwanlog/internal/app/usecase"
	"wanwanlog/internal/cli"
	"wanwanlog/internal/logrecord"
	"wanwanlog/internal/output"
	"wanwanlog/internal/parse"
	"wanwanlog/internal/search"
	"wanwanlog/internal/source"
	"wanwanlog/internal/tui"
	"wanwanlog/internal/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	config, err := cli.ParseArgs(os.Args[1:], os.Stderr)

	if err != nil {
		return err
	}

	if config.ShowHelp {
		printHelp()
		return nil
	}

	if config.ShowVersion {
		fmt.Println(version.Version)
		return nil
	}

	if config.TUI {
		return runTUI(config)
	}

	ctx := context.Background()

	store, err := search.OpenSQLiteStore(":memory:")
	if err != nil {
		return fmt.Errorf("create sqlite store: %w", err)
	}
	defer store.Close()

	uc := usecase.NewSearchLogsUseCase(
		func(path string) source.Source {
			return source.NewLocalFileSource(path)
		},
		parse.NewJSONLinesParser(),
		store,
	)

	result, err := uc.Execute(ctx, usecase.SearchLogsInput{
		InputPath: config.FilePath,
		Query:     config.Query,
		Limit:     100,
	})
	if err != nil {
		return err
	}

	var formatter output.Formatter

	if config.JsonFormatter {
		formatter = output.NewJSONFormatter(config.Pretty)
	} else {
		formatter = output.NewTableFormatter()
	}

	if err := formatter.Write(ctx, os.Stdout, output.SearchOutput{
		Query:   config.Query,
		Total:   result.Result.Total,
		Results: result.Result.Records,
	}); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

func runTUI(config cli.Config) error {
	ctx := context.Background()

	store, err := search.OpenSQLiteStore(":memory:")
	if err != nil {
		return fmt.Errorf("create sqlite store: %w", err)
	}
	defer store.Close()

	uc := usecase.NewSearchLogsUseCase(
		func(path string) source.Source {
			return source.NewLocalFileSource(path)
		},
		parse.NewJSONLinesParser(),
		store,
	)

	searchFunc := func(ctx context.Context, query string) ([]logrecord.LogRecord, error) {
		result, err := uc.Execute(ctx, usecase.SearchLogsInput{
			InputPath: config.FilePath,
			Query:     query,
			Limit:     100,
		})
		if err != nil {
			return nil, err
		}
		return result.Result.Records, nil
	}

	initialRecords, err := searchFunc(ctx, "")
	if err != nil {
		return err
	}

	attributeNames, err := collectAttributeNames(config.FilePath)
	if err != nil {
		return err
	}

	return tui.Run(ctx, searchFunc, initialRecords, tui.ModelOptions{
		AttributeNames:  attributeNames,
		SelectedColumns: config.Columns,
	})
}

func collectAttributeNames(path string) ([]string, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open input for column discovery: %w", err)
	}
	defer reader.Close()

	collector := tui.NewAttributeNameCollector()
	iterator := parse.NewJSONLinesParser().Parse(reader)
	for iterator.Next() {
		result := iterator.Result()
		if result.Err != nil {
			return nil, result.Err
		}
		collector.Add(result.Log.Record.Attributes)
	}
	if err := iterator.Err(); err != nil {
		return nil, fmt.Errorf("discover attribute columns: %w", err)
	}

	return collector.Names(), nil
}

func printHelp() {
	fmt.Println(`wwlog - Datadog like log viewer for local JSON Lines file

Usage:
  wwlog --file <path> [--query <query>] [--tui] [--columns <columns>]

Options:
  --file     Path to a JSON Lines log file
  --query    Search query
  --tui      Start in TUI mode
  --columns  Comma-separated TUI browse columns
  --json     Output as json
  --pretty   Pretty format
  --version  Show version
  --help     Show help`)
}

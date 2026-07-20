package main

import (
	"context"
	"fmt"
	"os"
	"wanwanlog/internal/app"
	"wanwanlog/internal/app/usecase"
	"wanwanlog/internal/cli"
	"wanwanlog/internal/output"
	"wanwanlog/internal/parse"
	"wanwanlog/internal/search"
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

	file, err := os.Open(config.FilePath)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	loader := usecase.NewLoadLogs(parse.NewJSONLinesParser(), store, 1000)
	if _, err := loader.Execute(ctx, file); err != nil {
		file.Close()
		return fmt.Errorf("load logs: %w", err)
	}
	_ = file.Close()
	uc := usecase.NewSearchLogsUseCase(store)

	result, err := uc.Execute(ctx, usecase.SearchLogsInput{
		Query: config.Query,
		Limit: 100,
	})
	if err != nil {
		return err
	}

	var formatter output.Formatter

	if config.JsonFormatter {
		formatter = output.NewJSONFormatter(config.Pretty)
	} else {
		formatter = output.NewTableFormatter(config.Columns...)
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

	application := &app.TUIApplication{
		Loader:   usecase.NewLoadLogs(parse.NewJSONLinesParser(), store, 1000),
		Searcher: usecase.NewSearchLogsUseCase(store),
		Store:    store,
	}
	return application.Run(ctx, app.TUIConfig{
		FilePath: config.FilePath, Query: config.Query, Columns: config.Columns, Follow: config.Follow,
	})
}

// collectAttributeNames remains as a small compatibility helper for callers that
// only need discovery. The TUI startup path collects these names during loading.
func collectAttributeNames(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open input for column discovery: %w", err)
	}
	defer file.Close()
	collector := usecase.NewAttributeNameCollector()
	iterator := parse.NewJSONLinesParser().Parse(file)
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
  wwlog --file <path> [--query <query>] [--columns <columns>]
  wwlog --file <path> --table [--query <query>] [--columns <columns>]
  wwlog --file <path> --json [--query <query>] [--pretty=false]

Options:
  --file     Path to a JSON Lines log file
  --query    Initial TUI filter or output search query
  --tui      Start in TUI mode (default)
  --table    Output as a table instead of starting the TUI
  --columns  Comma-separated table or TUI browse columns
  --json     Output as JSON instead of starting the TUI
  --pretty   Pretty-print JSON output (default true)
  --follow   Follow appended logs in TUI mode (default true)
  --version  Show version
  --help     Show help`)
}

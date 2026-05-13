package main

import (
	"context"
	"fmt"
	"os"
	"wanwanlog/internal/app/usecase"
	"wanwanlog/internal/cli"
	"wanwanlog/internal/output"
	"wanwanlog/internal/parse"
	"wanwanlog/internal/search"
	"wanwanlog/internal/source"
	"wanwanlog/internal/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// config, err := cli.ParseArgs(os.Args[1:], os.Stderr)

	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	os.Exit(1)
	// }

	// if config.ShowHelp {
	// 	printHelp()
	// 	return
	// }

	// if config.ShowVersion {
	// 	fmt.Println(version.Version)
	// 	return
	// }

	// src := source.NewLocalFileSource(config.FilePath)
	// reader, err := src.Open(context.Background())
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "failed to open source: %v\n", err)
	// 	os.Exit(1)
	// }
	// defer reader.Close()

	// parser := logparser.NewJSONLinesParser()

	// records, err := parser.Parse(reader)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "failed to parser logs: %v\n", err)
	// 	os.Exit(1)
	// }

	// for _, record := range records {
	// 	fmt.Printf("[%s] %s: %s\n", record.Level, record.Service, record.Message)
	// }
}

func run() error {
	config, err := cli.ParseArgs(os.Args[1:], os.Stderr)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if config.ShowHelp {
		printHelp()
		return nil
	}

	if config.ShowVersion {
		fmt.Println(version.Version)
		return nil
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

	fmt.Fprintf(os.Stderr, "total: %d\n", result.Result.Total)

	if err := output.WriteTable(os.Stdout, result.Result.Records); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

func printHelp() {
	fmt.Println(`wwlog - Datadog like log viewer for local JSON Lines file

Usage:
  wwlog --file <path> [--query <query>]

Options:
  --files    Path to a JSON Lines log file
  --query    Search query
  --version  Show version
  --help     Show help`)
}

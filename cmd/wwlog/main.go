package main

import (
	"context"
	"fmt"
	"os"
	"wanwanlog/internal/cli"
	"wanwanlog/internal/logparser"
	"wanwanlog/internal/source"
	"wanwanlog/internal/version"
)

func main() {
	config, err := cli.ParseArgs(os.Args[1:], os.Stderr)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if config.ShowHelp {
		printHelp()
		return
	}

	if config.ShowVersion {
		fmt.Println(version.Version)
		return
	}

	src := source.NewLocalFileSource(config.FilePath)
	reader, err := src.Open(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open source: %v\n", err)
		os.Exit(1)
	}
	defer reader.Close()

	parser := logparser.NewJSONLinesParser()

	records, err := parser.Parse(reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parser logs: %v\n", err)
		os.Exit(1)
	}

	for _, record := range records {
		fmt.Printf("[%s] %s: %s\n", record.Level, record.Service, record.Message)
	}
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

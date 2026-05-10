package main

import (
	"fmt"
	"os"
	"wanwanlog/internal/cli"
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

	fmt.Printf("file: %s\n", config.FilePath)

	if config.Query != "" {
		fmt.Printf("query: %s\n", config.Query)
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

package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

func ParseArgs(args []string, output io.Writer) (Config, error) {
	var config Config
	var columns string

	fs := flag.NewFlagSet("wwlog", flag.ContinueOnError)
	fs.SetOutput(output)

	fs.StringVar(&config.FilePath, "file", "", "Path to a JSON Lines log file")
	fs.StringVar(&config.Query, "query", "", "Search query")
	fs.StringVar(&columns, "columns", "", "Comma-separated output columns")
	fs.BoolVar(&config.TUI, "tui", false, "Start in TUI mode")
	fs.BoolVar(&config.JsonFormatter, "json", false, "Output as json")
	fs.BoolVar(&config.Pretty, "pretty", true, "Pretty format")
	fs.BoolVar(&config.ShowVersion, "version", false, "Show version")
	fs.BoolVar(&config.ShowHelp, "help", false, "Show help")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	columnsSet := false
	fs.Visit(func(flag *flag.Flag) {
		if flag.Name == "columns" {
			columnsSet = true
		}
	})

	if config.ShowHelp {
		return config, nil
	}

	if config.ShowVersion {
		return config, nil
	}

	if config.FilePath == "" {
		return Config{}, fmt.Errorf("--file is required")
	}

	if columnsSet {
		parsedColumns, err := parseColumns(columns)
		if err != nil {
			return Config{}, err
		}
		config.Columns = parsedColumns
	}

	return config, nil
}

func parseColumns(value string) ([]string, error) {
	parts := strings.Split(value, ",")
	columns := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		column := strings.TrimSpace(part)
		if !isValidColumn(column) {
			return nil, fmt.Errorf("invalid column %q", column)
		}
		if _, ok := seen[column]; ok {
			return nil, fmt.Errorf("duplicate column %q", column)
		}
		seen[column] = struct{}{}
		columns = append(columns, column)
	}

	return columns, nil
}

func isValidColumn(column string) bool {
	switch column {
	case "timestamp", "status", "service", "message":
		return true
	}

	if !strings.HasPrefix(column, "@") || len(column) == 1 {
		return false
	}
	for _, part := range strings.Split(column[1:], ".") {
		if part == "" {
			return false
		}
	}
	return true
}

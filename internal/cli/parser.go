package cli

import (
	"flag"
	"fmt"
	"io"
)

func ParseArgs(args []string, output io.Writer) (Config, error) {
	var config Config

	fs := flag.NewFlagSet("wwlog", flag.ContinueOnError)
	fs.SetOutput(output)

	fs.StringVar(&config.FilePath, "file", "", "Path to a JSON Lines log file")
	fs.StringVar(&config.Query, "query", "", "Search query")
	fs.BoolVar(&config.ShowVersion, "version", false, "Show version")
	fs.BoolVar(&config.ShowHelp, "help", false, "Show help")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if config.ShowHelp {
		return config, nil
	}

	if config.ShowVersion {
		return config, nil
	}

	if config.FilePath == "" {
		return Config{}, fmt.Errorf("--file is required")
	}

	return config, nil
}

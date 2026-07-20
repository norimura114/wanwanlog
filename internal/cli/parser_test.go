package cli

import (
	"bytes"
	"testing"
)

func TestParserArgsWithFile(t *testing.T) {
	var output bytes.Buffer

	config, err := ParseArgs([]string{
		"--file", "./logs/sample.jsonl",
	}, &output)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if config.FilePath != "./logs/sample.jsonl" {
		t.Fatalf("expected file path to be ./logs/sample.jsonl, got %s", config.FilePath)
	}

	if config.Query != "" {
		t.Fatalf("expected query to be empty, got %s", config.Query)
	}
	if !config.TUI {
		t.Fatal("expected TUI to be true by default")
	}
}

func TestParseArgsWithFileAndQuery(t *testing.T) {
	var output bytes.Buffer

	config, err := ParseArgs([]string{
		"--file", "./logs/sample.jsonl",
		"--query", "status:error",
	}, &output)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if config.FilePath != "./logs/sample.jsonl" {
		t.Fatalf("expected file path to be ./logs/sample.jsonl, got %s", config.FilePath)
	}

	if config.Query != "status:error" {
		t.Fatalf("expected query to be status:error, got %s", config.Query)
	}
}

func TestParseArgsWithTUI(t *testing.T) {
	var output bytes.Buffer

	config, err := ParseArgs([]string{
		"--file", "./logs/sample.jsonl",
		"--tui",
	}, &output)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !config.TUI {
		t.Fatal("expected TUI to be true")
	}
}

func TestParseArgsWithTableDisablesTUI(t *testing.T) {
	var output bytes.Buffer

	config, err := ParseArgs([]string{
		"--file", "./logs/sample.jsonl",
		"--table",
	}, &output)
	if err != nil {
		t.Fatal(err)
	}
	if config.TUI || !config.Table {
		t.Fatalf("config = %+v, want table mode", config)
	}
}

func TestParseArgsWithJSONDisablesTUI(t *testing.T) {
	var output bytes.Buffer

	config, err := ParseArgs([]string{
		"--file", "./logs/sample.jsonl",
		"--json",
	}, &output)
	if err != nil {
		t.Fatal(err)
	}
	if config.TUI || !config.JsonFormatter {
		t.Fatalf("config = %+v, want JSON mode", config)
	}
}

func TestParseArgsFollowDefaultsOnOnlyForTUI(t *testing.T) {
	var output bytes.Buffer
	tuiConfig, err := ParseArgs([]string{"--file", "logs.jsonl"}, &output)
	if err != nil {
		t.Fatal(err)
	}
	if !tuiConfig.Follow {
		t.Fatal("TUI should follow by default")
	}

	tableConfig, err := ParseArgs([]string{"--file", "logs.jsonl", "--table"}, &output)
	if err != nil {
		t.Fatal(err)
	}
	if tableConfig.Follow {
		t.Fatal("table output must remain a snapshot")
	}
}

func TestParseArgsRejectsExplicitFollowForSnapshotOutput(t *testing.T) {
	for _, outputFlag := range []string{"--table", "--json"} {
		var output bytes.Buffer
		if _, err := ParseArgs([]string{
			"--file", "logs.jsonl", outputFlag, "--follow=true",
		}, &output); err == nil {
			t.Fatalf("expected %s with --follow=true to fail", outputFlag)
		}
	}
}

func TestParseArgsRejectsConflictingOutputModes(t *testing.T) {
	tests := [][]string{
		{"--file", "logs.jsonl", "--table", "--json"},
		{"--file", "logs.jsonl", "--tui", "--table"},
		{"--file", "logs.jsonl", "--tui", "--json"},
	}

	for _, args := range tests {
		var output bytes.Buffer
		if _, err := ParseArgs(args, &output); err == nil {
			t.Fatalf("expected error for %v", args)
		}
	}
}

func TestParseArgsWithTUIColumns(t *testing.T) {
	var output bytes.Buffer

	config, err := ParseArgs([]string{
		"--file", "./logs/sample.jsonl",
		"--tui",
		"--columns", "timestamp,@http.status,message",
	}, &output)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := []string{"timestamp", "@http.status", "message"}
	if len(config.Columns) != len(want) {
		t.Fatalf("columns = %v, want %v", config.Columns, want)
	}
	for i := range want {
		if config.Columns[i] != want[i] {
			t.Fatalf("columns = %v, want %v", config.Columns, want)
		}
	}
}

func TestParseArgsAcceptsColumnsWithoutTUI(t *testing.T) {
	var output bytes.Buffer

	config, err := ParseArgs([]string{
		"--file", "./logs/sample.jsonl",
		"--table",
		"--columns", "timestamp,@http.status,message",
	}, &output)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := []string{"timestamp", "@http.status", "message"}
	for i := range want {
		if config.Columns[i] != want[i] {
			t.Fatalf("columns = %v, want %v", config.Columns, want)
		}
	}
}

func TestParseArgsRejectsInvalidAndDuplicateColumns(t *testing.T) {
	tests := []string{
		"timestamp,level",
		"timestamp,timestamp",
		"timestamp,@",
		"timestamp,@http..status",
		"",
	}

	for _, columns := range tests {
		t.Run(columns, func(t *testing.T) {
			var output bytes.Buffer
			_, err := ParseArgs([]string{
				"--file", "./logs/sample.jsonl",
				"--tui",
				"--columns", columns,
			}, &output)
			if err == nil {
				t.Fatalf("expected error for columns %q", columns)
			}
		})
	}
}

func TestParseArgsWithoutFile(t *testing.T) {
	var output bytes.Buffer

	_, err := ParseArgs([]string{}, &output)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseArgsWithHelp(t *testing.T) {
	var output bytes.Buffer

	config, err := ParseArgs([]string{"--help"}, &output)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !config.ShowHelp {
		t.Fatal("expected ShowHelp to be true")
	}
}

func TestParseArgsWithVersion(t *testing.T) {
	var output bytes.Buffer

	config, err := ParseArgs([]string{"--version"}, &output)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !config.ShowVersion {
		t.Fatal("expected ShowVersion to be true")
	}
}

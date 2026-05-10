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

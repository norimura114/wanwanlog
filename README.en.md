# wanwanlog

[日本語](README.md)

```text
(ᐡ ᐧ ﻌ ᐧ ᐡ) Dig through logs, wanwan!
```

A terminal-based log viewer for browsing and searching local JSON Lines files.
It provides a Datadog-like search syntax, an interactive TUI, table output, and JSON output.

> [!WARNING]
> This application was created through vibe coding with ChatGPT.
> Its quality, safety, and reliability are not guaranteed. Do not use it with
> critical data or in production environments. Use it at your own risk.

> [!NOTE]
> This project is currently in alpha. Options and search syntax may change in the future.

## Features

- Read local JSON Lines files directly
- Browse logs interactively using Bubble Tea
- Search fields and attributes with Datadog-like syntax
- Select and reorder displayed columns
- Produce non-interactive table or JSON output
- Process data locally without sending it to external services

## Installation

Download the archive for your operating system and architecture from
[Releases](https://github.com/norimura114/wanwanlog/releases).

Building from source requires Go 1.26.3 or later.

```bash
git clone https://github.com/norimura114/wanwanlog.git
cd wanwanlog
go build -o wwlog ./cmd/wwlog
```

## Usage

Specify a JSON Lines file with `--file`. The TUI starts by default.

```bash
wwlog --file app.jsonl
```

You can also specify an initial filter and the columns to display.

```bash
wwlog \
  --file app.jsonl \
  --query 'service:api status:error' \
  --columns 'timestamp,status,service,message'
```

### Table output

Use `--table` when running wanwanlog from a script or pipeline.

```bash
wwlog \
  --file app.jsonl \
  --table \
  --query '@http.status_code >= 500' \
  --columns 'timestamp,service,@http.status_code,message'
```

### JSON output

```bash
wwlog --file app.jsonl --json --query 'status:error'
```

To produce compact JSON without indentation:

```bash
wwlog --file app.jsonl --json --pretty=false
```

### Options

```text
--file <path>       Path to a JSON Lines file (required)
--query <query>     Initial TUI filter or search query for non-interactive output
--tui               Explicitly start the TUI (default)
--table             Output in table format
--columns <list>    Comma-separated list of columns to display
--json              Output in JSON format
--pretty            Pretty-print JSON (default: true)
--version           Display the version
--help              Display help
```

## Search syntax

Adjacent conditions are treated as `AND`.

```text
service:api status:error
```

Supported search forms:

```text
service:api                 Field match
message:"request timeout"   Phrase match
@http.status_code:500       Attribute match
@http.duration_ms >= 1000   Numeric attribute comparison
timeout                     Keyword search in message
-status:info                Negation
```

The built-in fields are `timestamp`, `service`, `status`, and `message`.
Reference attributes using dot-separated paths such as `@http.status_code`.

`OR` and parenthesized expressions are not currently supported.

## Input format

Each line must contain one JSON object.

```json
{"timestamp":"2026-05-10T12:00:00Z","service":"api","status":"error","message":"request timeout","http":{"status_code":504,"duration_ms":3000}}
```

`timestamp`, `service`, `status`, and `message` are treated as built-in fields
for the list view. All other values are retained as attributes and can be used
in searches or displayed as columns. Nested objects are supported.

Blank lines are ignored, and the maximum line size is 10 MiB. The entire file
is loaded into an in-memory SQLite database on each startup.

## TUI key bindings

```text
j / ↓       Move down
k / ↑       Move up
g / G       Jump to first / last
Enter / l   Show details
h / Esc     Return to list
/           Edit filter
c           Edit displayed columns
?           Show help
q           Quit
```

## Development

```bash
go test ./...
go vet ./...
```

To regenerate the test logs:

```bash
go run ./cmd/gen-tetdata \
  --output testdata/logs.jsonl \
  --count 1000 \
  --seed 1
```

To manually test appended-log loading, append logs at regular intervals from
another terminal. `--count 0` continues until you press Ctrl+C.

```bash
go run ./cmd/gen-tetdata \
  --output testdata/logs.jsonl \
  --follow \
  --count 0 \
  --interval 500ms
```

See the [design document](docs/design.md) for implementation details.

Releases are automated with Release Please and GoReleaser. Commit messages
follow [Conventional Commits](https://www.conventionalcommits.org/).

## License

[MIT License](LICENSE)

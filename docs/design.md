# wanwanlog Design

This document describes the current implementation design of `wanwanlog`.

`wanwanlog` is a local JSON Lines log viewer. The current CLI flow reads a JSON Lines file, parses each line into a normalized log record, loads the records into an in-memory SQLite database, translates a Datadog-like filter string into SQL, and prints matching logs as a table or JSON.

## Goals

- View and search local JSON Lines log files without an external log backend.
- Keep parsing, storage, filtering, and presentation separated behind small package boundaries.
- Preserve the original JSON object as both structured attributes and raw text.
- Support a query style close to Datadog log search for common local workflows.

## High-Level Flow

```text
cmd/wwlog
  -> internal/cli
  -> internal/app/usecase.SearchLogsUseCase
       -> internal/source.Source
       -> internal/parse.Parser
       -> internal/search.Store
            -> internal/filter.Parse
            -> internal/filter.SQLBuilder
  -> internal/output.Formatter
```

At runtime, `cmd/wwlog/main.go` wires the concrete dependencies:

- `source.NewLocalFileSource(path)` opens the local file.
- `parse.NewJSONLinesParser()` streams JSON Lines into `logrecord.LogRecord` values.
- `search.OpenSQLiteStore(":memory:")` creates the transient search store.
- `output.NewTableFormatter()` or `output.NewJSONFormatter(pretty)` renders the result.

The SQLite database is currently in-memory, so each command invocation reloads the input file before searching.

## Package Responsibilities

### `cmd/wwlog`

The command package is the composition root. It parses CLI arguments, creates concrete implementations, invokes the search use case, and writes output.

Current user-facing flags:

- `--file`: JSON Lines input file path.
- `--query`: initial TUI filter or non-interactive filter expression.
- `--tui`: explicitly start the Bubble Tea TUI, which is the default mode.
- `--table`: print a table instead of starting the TUI.
- `--columns`: comma-separated table columns or initial TUI browse columns.
- `--json`: output as JSON instead of table.
- `--pretty`: pretty-print JSON output.
- `--version`: print version.
- `--help`: print help.

By default, the command loads logs into the in-memory store and starts the
Bubble Tea interface. `--table` and `--json` select non-interactive output
instead. The output mode flags are mutually exclusive.

### `internal/cli`

`internal/cli` owns command-line parsing and validation. It returns a `Config` struct and keeps flag parsing independent from the command entrypoint.

Validation is intentionally small: `--file` is required unless `--help` or `--version` is requested.

### `internal/app/usecase`

The use case layer coordinates application workflow and depends on interfaces rather than concrete I/O details.

`SearchLogsUseCase` performs the main flow:

1. Validate input path and normalize limit.
2. Initialize the store.
3. Open the source.
4. Parse and insert records in batches.
5. Execute a store search.
6. Return a `filter.SearchResult`.

`LoadLogs` is a separate batching use case that parses from an `io.Reader` and inserts logs into a store while counting parse errors. The current CLI uses the private loading logic inside `SearchLogsUseCase` instead.

### `internal/source`

The source package abstracts where log bytes come from.

```go
type Source interface {
    Open(ctx context.Context) (io.ReadCloser, error)
}
```

The current implementation is `LocalFileSource`, which opens a file from disk. This boundary leaves room for future sources such as stdin, remote files, or compressed files without changing parsing and search code.

### `internal/parse`

The parse package converts input streams into log records.

```go
type Parser interface {
    Parse(r io.Reader) Iterator
}

type Iterator interface {
    Next() bool
    Result() ParseResult
    Err() error
}
```

`JSONLinesParser` scans the reader line by line with a 10 MiB maximum line size. Empty lines are skipped. Each non-empty line is unmarshaled as JSON into `map[string]any`.

The parser maps common fields onto first-class `LogRecord` fields:

- `id`
- `timestamp`
- `message`
- `service`
- `status`

The full JSON object is also stored in `Attributes`, and the original line is stored in `Raw`.

If no `id` is present, the parser generates a random 16-byte hex ID. If the timestamp cannot be parsed as RFC3339 or RFC3339Nano, it is left as the zero time.

### `internal/logrecord`

`LogRecord` is the central domain data type:

```go
type LogRecord struct {
    ID         string
    Timestamp  time.Time
    Message    string
    Service    string
    Status     string
    Attributes map[string]any
    Raw        string
}
```

The design keeps frequently queried fields as typed top-level fields while preserving the entire JSON payload in `Attributes` for flexible attribute search and output.

### `internal/search`

The search package defines the storage boundary and provides the SQLite implementation.

```go
type Store interface {
    Init(ctx context.Context) error
    InsertBatch(ctx context.Context, records []logrecord.LogRecord) error
    Search(ctx context.Context, query filter.SearchQuery) (filter.SearchResult, error)
    Close() error
}
```

`SQLiteStore` stores logs in a single `logs` table:

```sql
CREATE TABLE IF NOT EXISTS logs (
  id TEXT PRIMARY KEY,
  timestamp TEXT,
  service TEXT,
  status TEXT,
  message TEXT,
  attributes TEXT NOT NULL,
  raw TEXT NOT NULL
);
```

Indexes are created for `timestamp`, `service`, and `status`.

Attributes are serialized as JSON text and queried using SQLite JSON functions. Search results are ordered by `timestamp ASC`, limited by `SearchQuery.Limit`, and offset by `SearchQuery.Offset`.

### `internal/filter`

The filter package owns query parsing and SQL generation.

The query parser currently supports implicit `AND` between adjacent terms:

```text
service:api status:error @http.status >= 400 timeout
```

Supported expression shapes:

- Field equality: `service:api`, `status:error`, `message:"some text"`.
- Attribute comparisons: `@user.id:123`, `@http.status >= 400`.
- Keyword search: `timeout`.
- Negation: `-status:info`.
- Empty query: matches all records.

Supported field names are `timestamp`, `service`, `status`, and `message`.

Attribute keys support dotted JSON paths such as `@user.id`. Attribute literals are parsed as strings, numbers, booleans, or null-like values, and numeric range comparisons are only accepted for number literals.

`SQLBuilder` converts the AST into a parameterized SQL `WHERE` clause:

- Field expressions become direct column equality.
- Keyword expressions become `message LIKE ? ESCAPE '\'`.
- Attribute expressions use `json_type` and `json_extract`.
- Multiple top-level expressions are joined with `AND`.
- Negated expressions are wrapped with `NOT (...)`.

### `internal/output`

The output package formats search results.

`TableFormatter` prints a compact table with time, level, service, message, and
flattened attributes. When `--columns` is provided, it instead prints the
selected built-in fields and attribute paths as independent columns. Long
message and attribute strings are truncated.

`JSONFormatter` writes the `SearchOutput` struct as JSON and supports indentation through the `Pretty` flag.

### `internal/tui`

The TUI package implements a Bubble Tea model with these modes:

- `BrowseMode`: list logs and move selection.
- `FilterMode`: edit and execute a filter query.
- `DetailMode`: inspect the selected record.
- `HelpMode`: show key bindings.
- `ColumnsMode`: choose, show or hide, and reorder browse columns.

The TUI receives a `SearchFunc`, so UI state management is separated from log
loading and query execution. It is the default command mode. When `--query` is
provided, the TUI opens with that filter already applied.

The browse list supports the built-in `timestamp`, `status`, `service`, and
`message` columns plus attribute paths such as `@http.status`. Available
attribute paths are discovered from the full input file when the TUI starts.
Attributes that only contain arrays or objects are omitted from the column
settings; nested scalar paths remain available.
Column configuration is kept only for the lifetime of the TUI process.

## Filter Grammar

The effective grammar is intentionally simple:

```text
query      := expr*
expr       := "-" expr
           | "@" attribute_key compare_op literal
           | ident ":" value
           | ident
           | quoted_string

compare_op := ":" | "=" | "!=" | ">" | ">=" | "<" | "<="
value      := ident | quoted_string
literal    := ident | quoted_string
```

Adjacent expressions are combined with `AND`. `OR` and parentheses are tokenized in the lexer but are not currently handled by the parser.

## Data Lifecycle

1. The source opens an `io.ReadCloser`.
2. The parser returns an iterator over parsed records and per-line parse errors.
3. The use case consumes the iterator until exhausted or context cancellation.
4. Records are accumulated into batches of 1000 by default.
5. Each batch is inserted in a single SQLite transaction.
6. The search store parses the filter, builds SQL, counts total matches, then fetches the current page of records.
7. The formatter writes the result to stdout.

For `SearchLogsUseCase`, the first parse error aborts the search. For `LoadLogs`, parse errors are counted while valid records continue to load.

## Current Constraints and Notes

- Storage is transient. The default CLI always uses `:memory:` and reloads the full file for every query.
- There is no pagination flag in the CLI even though the store supports `Limit` and `Offset`.
- The filter lexer has tokens for `OR` and parentheses, but the parser currently implements only implicit `AND` and unary negation.
- Attribute search relies on SQLite JSON functions over the serialized `attributes` column.
- `service`, `status`, `message`, and `timestamp` are duplicated between top-level columns and the JSON `attributes` payload.
- The codebase has some implementation mismatches worth resolving before expanding behavior:
  - Table time formatting uses `2006-01-012 15:04:05`, which appears to include an extra `2`.

## Extension Points

- Add new sources by implementing `source.Source`.
- Add new input formats by implementing `parse.Parser`.
- Add persistent or alternative search backends by implementing `search.Store`.
- Extend query behavior by adding AST nodes in `internal/filter`, parser support, and SQL builder support.

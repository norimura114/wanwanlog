# Live File Follow

- Status: Proposed
- Date: 2026-07-20

## Background

現在の TUI は起動時とフィルター実行時に入力ファイル全体を読み直す。起動後に
ファイルへ追記されたログは自動反映されず、フィルターを再実行すると既にロードした
レコードを再挿入する可能性もある。

TUI を終了せず、JSON Lines ファイルへ追記されたログを継続的に閲覧できるようにする。

## Scope

- 対象はローカルファイルを入力とする TUI モードとする。
- 起動時に既存内容をロードし、その後の追記を自動的に読み込む。
- 現在のフィルターを新しいログにも適用する。
- ファイルの truncate と rename による一般的なログローテーションに対応する。
- `--table` と `--json` は従来どおり有限のスナップショット出力とする。

## User-visible Behavior

`wwlog --file app.jsonl` は初期ロード完了後もファイルを監視する。完全な JSON 行が
追記されると、既定では 500 ms 以内を目安に一覧へ反映する。

一覧が末尾を選択しているときは、新しい一致レコードの追加後も末尾を選択する。
過去のレコードを選択しているとき、detail pane を表示しているとき、フィルターや
カラムを編集中のときは、選択中のレコードと画面モードを維持し勝手に末尾へ移動しない。

ヘッダーに次の状態を表示する。

- `live`: 追記を監視中
- `loading`: 初期ロード中
- `retrying`: ファイルの再オープン待ち
- `error`: 回復不能なエラー

追記された行が現在のフィルターに一致しない場合、一覧は変化しない。フィルター変更は
SQLite 内の既ロードデータだけを検索し、入力ファイルを読み直さない。

## Prerequisite Refactoring

追記監視を追加する前に、既存動作を変えないリファクタリングを行う。監視処理と既存の
責務整理を同時に実施すると、ファイル読み込み、SQLite、TUI の非同期更新にまたがる変更と
なり、障害原因を切り分けにくくなるためである。

### 1. Separate Loading from Searching

現在の `SearchLogsUseCase.Execute` は、呼び出すたびに入力ファイルを開き、全件を SQLite
へ投入してから検索する。TUI のフィルター実行もこの経路を使うため、検索とファイル
読み込みが分離されていない。

既存の `LoadLogs` を正式なロード処理として採用し、`SearchLogsUseCase` 内の private な
`load` 実装を削除する。application layer の境界を次のように分ける。

```go
type LogLoader interface {
    Load(ctx context.Context, r io.Reader) (LoadResult, error)
}

type LogSearcher interface {
    Search(
        ctx context.Context,
        query filter.SearchQuery,
    ) (filter.SearchResult, error)
}
```

- `LogLoader` は parse、batch insert、ロード結果の集計だけを担当する。
- `LogSearcher` は既に store に存在するレコードの検索だけを担当する。
- フィルター変更では `LogSearcher` だけを呼び、入力ファイルを開かない。
- 初期ロードを複数回実行しなくても従来と同じ検索結果になることをテストする。

この変更を追記監視実装より先に完了させる。

### 2. Collect Attributes During Loading

現在の `collectAttributeNames` は、初期ロードとは別に入力ファイル全体を再度読み込む。
attribute 候補の収集を `LogLoader` の処理へ統合し、各行を一度だけ parse する。

```go
type LoadResult struct {
    LoadedCount    int
    ErrorCount     int
    AttributeNames []string
}
```

初期ロード時は `AttributeNames` を TUI のカラム候補へ渡す。追記監視の実装後は、同じ
collector に追記レコードを渡して新しい候補を検出する。

collector は application または domain service として配置し、TUI パッケージにある
表示用の `AttributeNameCollector` をロード処理から直接参照しない。TUI は収集済みの
attribute 名を受け取るだけにする。

### 3. Return a Structured TUI Search Result

現在の TUI の `SearchFunc` は `[]logrecord.LogRecord` だけを返す。総件数、最新 window、
検索 generation を追加できるよう、先に結果型を導入する。

```go
type SearchResult struct {
    Records []logrecord.LogRecord
    Total   int
}

type SearchFunc func(
    ctx context.Context,
    query string,
) (SearchResult, error)
```

この段階では follow event、`dirty`、generation の動作は追加せず、既存の同期的な
フィルター操作と表示を維持する。`searchDoneMsg` も `SearchResult` を保持する形へ変更する。

### 4. Make Search Ordering Explicit

現在の store 検索は timestamp 昇順と固定 limit を暗黙に使用する。live mode では
「最新 N 件を取得して画面内では昇順に表示する」操作が必要になるため、検索条件へ
明示的な順序を追加する。

```go
type SearchOrder int

const (
    OldestFirst SearchOrder = iota
    LatestWindow
)

type SearchQuery struct {
    Filter string
    Limit  int
    Offset int
    Order  SearchOrder
}
```

リファクタリング時点のデフォルトは `OldestFirst` とし、既存の table、JSON、TUI の
結果順序を変更しない。`LatestWindow` の利用開始は follow 実装時とする。

同一 timestamp と timestamp 欠落時にも安定した順序になるよう、SQLite 内部の挿入順を
tie breaker として利用できる schema にする。既存の `id` の意味は変更しない。

### 5. Extract TUI Application Orchestration

現在の `cmd/wwlog/main.go` は SQLite の生成、use case の構築、初期検索、attribute 収集、
TUI 用 closure の生成、リソース解放を担当する。追記監視では follower の開始と停止、
context cancellation、TUI 終了順も加わるため、TUI の実行調停を application layer へ
移す。

```go
type TUIApplication struct {
    Loader   LogLoader
    Searcher LogSearcher
    Store    search.Store
}

func (a *TUIApplication) Run(
    ctx context.Context,
    config Config,
) error
```

具体的な concrete dependency の生成は composition root である `cmd/wwlog` に残す。
`TUIApplication` は生成済み依存を受け取り、初期ロード、TUI 実行、終了処理を調停する。
follow 実装時に `Follower` をこの application へ追加する。

### Refactoring Sequence

リファクタリングは次の順序で行い、各段階で `go test ./...` を成功させる。

1. `LoadLogs` へロード処理を一本化する。
2. attribute 候補収集をロード処理へ統合する。
3. TUI の `SearchFunc` を構造化された結果型へ変更する。
4. store の検索順序を明示し、既存順序を維持する。
5. TUI 起動処理を `TUIApplication` へ移す。
6. その後に `LocalFileFollower` と follow event を実装する。

1〜5では、追記監視、polling、goroutine、ファイルローテーション処理を導入しない。
各変更を既存動作のまま独立して検証できる状態を保つ。

## Architecture

処理を次の3つに分離する。

```text
LocalFileFollower
  -> LogIngestor
       -> Parser
       -> Search Store
  -> TUI: data changed event
       -> Search Service
       -> Model update
```

### Initial Load

1. ファイルを開き、ファイル識別情報と現在の offset を管理する。
2. 起動時点の EOF まで完全な行を読み、SQLite へバッチ挿入する。
3. EOF 到達後に TUI を起動し、同じ follower が追記監視へ移行する。
4. カラム候補は初期ロードしたレコードから収集する。

初期ロードと追記処理は同一の ingest 経路を使い、ファイルを読む主体は常に1つだけにする。

### Follow Loop

`internal/source` に `LocalFileFollower` を追加する。OS 固有の通知 API には依存せず、
最初の実装は 500 ms 間隔の polling とする。これにより macOS、Linux、Windows で
同じ動作を保つ。

follower は次を保持する。

- 開いているファイル
- ファイル識別情報
- 次に読む byte offset
- 改行前の未確定 byte buffer

各 poll では現在のファイル状態を取得し、サイズが offset より大きければ追加分だけ読む。
追記データは `\n` 単位で切り出し、改行で完結した行だけを parser へ渡す。EOF にある
改行前の断片はエラーにせず、次回の追記まで buffer に保持する。空行は従来どおり無視する。

polling は `context.Context` のキャンセルで停止する。TUI 終了時は context を cancel し、
goroutine、ticker、ファイル descriptor を確実に閉じる。

### Ingestion

`SearchLogsUseCase` の「ロードして検索する」責務を分割する。

- `LogIngestor.InitialLoad(ctx)`: 既存内容を一度だけ投入する。
- `LogIngestor.Append(ctx, lines)`: 完全な追記行を解析しバッチ投入する。
- `SearchService.Search(ctx, query, limit)`: SQLite のみを検索する。

追記行は1回だけ解析・挿入する。検索操作はファイルへアクセスしない。

1回の poll で得た完全な行を1トランザクションで挿入し、commit 成功後にのみ
`DataChanged` を通知する。途中の行に不正な JSON があっても follower は停止せず、
その行をスキップしてエラー件数を通知する。初期ロード時も同じ方針に統一する。

ID が入力にないレコードは、初回の解析時にだけ ID を生成する。offset を正しく管理し、
同じ行の再解析による重複を防ぐ。

### TUI Integration

TUI には継続イベントを直接送る channel を持たせず、Bubble Tea の command/subscription
として受け取る。

```go
type FollowEvent struct {
    AddedCount      int
    ParseErrorCount int
    Reset           bool
    Err             error
}
```

`Model.Init` は次の follower event を待つ command を返す。イベント受信後は、
現在の `filterText` で非同期検索を行い、同時に次の event 待受 command を登録する。

短時間に追記が集中した場合は、検索中に届いた更新を `dirty` フラグへまとめる。
進行中の検索完了後、`dirty` なら1回だけ再検索する。古い検索結果が新しい結果を
上書きしないよう、検索 request に単調増加する generation を付ける。

結果適用時は選択中レコードの ID を使って位置を復元する。更新前に末尾を選択していた
場合だけ、新しい結果の末尾を選択する。選択レコードがフィルター結果から消えた場合は、
直前の index に最も近い行を選択する。

追記で新しい scalar attribute が登場した場合はカラム候補へ追加する。既存の表示設定を
変えず、新しい候補は非表示で追加する。ただし `--columns` で指定済みなら表示状態を保つ。

## Search Window

live mode の一覧は「現在の条件に一致する最新100件」を表示し、画面内では timestamp の
昇順に並べる。SQLite では内側の query を `timestamp DESC, rowid DESC LIMIT 100` とし、
取得後に昇順へ戻す。これにより100件を超えるファイルでも新しいログが一覧へ現れる。

同一 timestamp や timestamp 欠落時も順序を安定させるため、store 内部の挿入順を表す
連番を用いる。入力 JSON の `id` は重複排除には使うが、表示順には使わない。

非 TUI 出力の順序と limit はこの変更の対象外とする。

## Rotation and Truncation

各 poll で、path が指すファイルと開いているファイルの識別情報およびサイズを比較する。

### copytruncate

同じファイルのサイズが現在の offset より小さくなった場合:

1. offset と未確定 buffer を 0 に戻す。
2. SQLite の既存ログを削除する。
3. 切り詰め後のファイルを先頭から再ロードする。
4. `Reset` event を TUI へ送る。

画面は新しいファイル内容だけを表示する。

### rename and recreate

path のファイル識別情報が開いているファイルと異なる場合:

1. 旧 file descriptor に残る EOF までを読む。
2. 新しい path を開く。
3. offset と未確定 buffer を 0 に戻す。
4. SQLite の既存ログを削除する。
5. 新ファイルを先頭からロードし、`Reset` event を送る。

ローテーションをまたいだ履歴保持は初期実装では行わない。

path が一時的に存在しない場合は follower を終了せず、指数 backoff
（500 ms から最大5秒）で再オープンする。直近の一覧は維持し、状態を `retrying` とする。

## Error Handling

- 書きかけ行: 改行まで保留し、エラー表示しない。
- 不正 JSON の完全な行: 行をスキップし、累積 parse error 数をヘッダーへ表示する。
- 10 MiB を超える行: その行をスキップし、parse error として扱う。
- 一時的な stat/open/read エラー: 一覧を維持して retry する。
- SQLite insert/search エラー: `error` として TUI に表示し、入力監視は継続する。
- context cancellation: 正常終了として扱う。

## Interfaces

新規または変更する主要な境界は次のとおり。

```go
type Follower interface {
    InitialRead(ctx context.Context) (Batch, error)
    Events() <-chan FollowEvent
    Run(ctx context.Context)
    Close() error
}

type SearchService interface {
    Search(ctx context.Context, query string, limit int) ([]logrecord.LogRecord, error)
}
```

具体的な API は実装時に Go の ownership を単純にできる形へ調整してよいが、
「ファイルを読む主体は1つ」「store は ingestor と search で共有」
「TUI は FollowEvent だけを受け取る」という境界は維持する。

store には次の操作を追加する。

- `DeleteAll(ctx)`: truncate/rotation 時の reset
- `SearchLatest(ctx, query, limit)`: 最新 N 件を安定順序で取得

## CLI

TUI の local file はデフォルトで follow する。明示的に固定スナップショットを開けるよう、
`--follow` を bool flag として追加し、TUI では既定値を true とする。

```console
wwlog --file app.jsonl
wwlog --file app.jsonl --follow=false
```

`--table` と `--json` では `--follow=true` をエラーにする。将来の streaming output は
別仕様とする。

## Tests

### Unit tests

- 初期 EOF 後の追記行だけを1回返す。
- 複数 poll に分かれた書きかけ行を改行後に1回だけ返す。
- 1 poll 内の複数行を1 batch として挿入する。
- 不正 JSON 行をスキップし、次の正常行を読み込む。
- truncate を検出し、offset、buffer、store を reset する。
- rename/recreate を検出し、新ファイルを先頭から読む。
- path 消失中に retry し、再作成後に回復する。
- context cancel 後に goroutine と file descriptor が残らない。
- 追記イベントの coalescing と検索 generation の競合を検証する。
- 過去行の選択維持と、末尾選択時の自動追従を検証する。
- 新しい attribute が非表示カラム候補として追加される。

### Integration test

一時ファイルで TUI の下位サービスを起動し、初期行、分割追記、通常追記、
copytruncate、rename/recreate の順に操作して、最終的な store 内容と event 列を検証する。
polling interval は注入可能にし、テストでは短い interval を使う。

race detector を含めて検証する。

```console
go test -race ./...
```

## Acceptance Criteria

- TUI 起動後に追記された完全な JSON 行が、再起動や手動再検索なしで表示される。
- 書き込み途中の JSON は一時的な parse error にならない。
- 100件を超える入力でも最新ログが表示される。
- 過去ログの閲覧中に新規追記があっても選択位置が移動しない。
- filter 操作でファイル全体を再読込せず、重複レコードを作らない。
- truncate と rename/recreate 後に、新しいファイル内容へ自動的に切り替わる。
- TUI 終了後に監視 goroutine と file descriptor が残らない。
- `go test -race ./...` が成功する。

## Out of Scope

- `--table` / `--json` のストリーミング出力
- ローテーション済みファイルを横断した履歴保持
- 複数ファイルや glob の同時監視
- stdin、圧縮ファイル、リモート source の follow
- 監視間隔の CLI 設定
- SQLite データのプロセス間永続化

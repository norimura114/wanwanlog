# wanwanlog

ローカルの JSON Lines ログを、ターミナル上で閲覧・検索するためのログビューアです。
Datadog に近い検索構文、対話的な TUI、テーブル出力、JSON 出力を提供します。

> [!NOTE]
> 現在はアルファ版です。オプションや検索構文は今後変更される可能性があります。

## Features

- ローカルの JSON Lines ファイルを直接読み込み
- Bubble Tea を使用した対話的なログ閲覧
- Datadog ライクなフィールド・属性検索
- 表示カラムの選択、並べ替え
- テーブルまたは JSON による非対話出力
- 入力データを外部サービスへ送信しないローカル処理

## Installation

[Releases](https://github.com/norimura114/wanwanlog/releases) から、お使いの
OS・アーキテクチャ向けのアーカイブをダウンロードしてください。

ソースからビルドする場合は Go 1.26.3 以降が必要です。

```bash
git clone https://github.com/norimura114/wanwanlog.git
cd wanwanlog
go build -o wwlog ./cmd/wwlog
```

## Usage

`--file`に JSON Lines ファイルを指定すると、デフォルトで TUI が起動します。

```bash
wwlog --file app.jsonl
```

初期フィルターや表示カラムも指定できます。

```bash
wwlog \
  --file app.jsonl \
  --query 'service:api status:error' \
  --columns 'timestamp,status,service,message'
```

### Table output

スクリプトやパイプラインから利用する場合は`--table`を指定します。

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

インデントなしの JSON を出力する場合:

```bash
wwlog --file app.jsonl --json --pretty=false
```

### Options

```text
--file <path>       JSON Linesファイルのパス（必須）
--query <query>     TUIの初期フィルターまたは出力時の検索条件
--tui               TUIを明示的に起動（デフォルト）
--table             テーブル形式で出力
--columns <list>    表示カラムをカンマ区切りで指定
--json              JSON形式で出力
--pretty            JSONを整形して出力（デフォルト: true）
--version           バージョンを表示
--help              ヘルプを表示
```

## Search syntax

隣接する条件は`AND`として扱われます。

```text
service:api status:error
```

利用できる検索形式:

```text
service:api                 フィールド一致
message:"request timeout"   フレーズ一致
@http.status_code:500       属性一致
@http.duration_ms >= 1000   属性の数値比較
timeout                     messageのキーワード検索
-status:info                否定
```

組み込みフィールドは`timestamp`、`service`、`status`、`message`です。
属性は`@http.status_code`のようなドット区切りのパスで参照します。

`OR`と括弧を使った条件は、現在未対応です。

## Input format

1行に1つの JSON オブジェクトを記述します。

```json
{"timestamp":"2026-05-10T12:00:00Z","service":"api","status":"error","message":"request timeout","http":{"status_code":504,"duration_ms":3000}}
```

`timestamp`、`service`、`status`、`message`は一覧表示用の組み込みフィールドとして
扱われます。それ以外の値も属性として保持され、検索やカラム表示に利用できます。
ネストしたオブジェクトにも対応しています。

空行は無視され、1行の最大サイズは10 MiBです。起動ごとにファイル全体を
インメモリのSQLiteデータベースへ読み込みます。

## TUI key bindings

```text
j / ↓       下へ移動
k / ↑       上へ移動
g / G       先頭 / 末尾へ移動
Enter / l   詳細を表示
h / Esc     一覧へ戻る
/           フィルターを編集
c           表示カラムを編集
?           ヘルプ
q           終了
```

## Development

```bash
go test ./...
go vet ./...
```

テスト用ログを再生成する場合:

```bash
go run ./cmd/gen-tetdata \
  --output testdata/logs.jsonl \
  --count 1000 \
  --seed 1
```

追記読み込みを手動確認する場合は、別のターミナルから一定間隔でログを追記できます。
`--count 0` は Ctrl+C まで継続します。

```bash
go run ./cmd/gen-tetdata \
  --output testdata/logs.jsonl \
  --follow \
  --count 0 \
  --interval 500ms
```

詳細な実装については[設計ドキュメント](docs/design.md)を参照してください。

リリースはRelease PleaseとGoReleaserで自動化しています。コミットメッセージには
[Conventional Commits](https://www.conventionalcommits.org/)を使用します。

## License

[MIT License](LICENSE)

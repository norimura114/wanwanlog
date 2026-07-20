# Browse Pane Column Customization

- Status: Approved
- Date: 2026-07-20

## Background

TUI の browse pane では、現在 `timestamp`、`status`、`service`、`message` が固定順で表示される。利用するログによって確認したいフィールドが異なるため、一覧に表示するカラムをカスタマイズできるようにする。

## Scope

- 対象は TUI 左側の browse pane の一覧のみとする。
- detail pane および非 TUI のテーブル出力は変更しない。
- カラム設定は実行中のみ保持し、ファイルや設定ファイルには保存しない。

## Default Columns

デフォルトでは、次のカラムをこの順番で表示する。

1. `timestamp`
2. `status`
3. `service`
4. `message`

一覧の先頭にはカラム名を示すヘッダー行を表示する。

## Available Columns

組み込みカラムと、入力ファイル内の任意の attribute を選択できる。

組み込みカラムは次のとおりとする。

- `timestamp`
- `status`
- `service`
- `message`

attribute の選択候補は、入力ファイル全体に登場するキーの和集合から生成する。ネストしたキーは `@http.status` のようなドット表記で扱う。

あるレコードに選択した attribute が存在しない場合、そのセルは空欄で表示する。

attribute の値が配列の場合は `[array]`、オブジェクトの場合は `{object}` と表示し、内容は一覧上では展開しない。配列やオブジェクトの attribute もカラムとして選択できる。

## TUI Configuration Screen

browse mode で `c` を押すと、カラム設定画面を開く。

設定画面では次のキー操作を提供する。

- `j` / `k`: カラムを選択
- `space`: 表示・非表示を切り替え
- `J` / `K`: カラムの表示順を変更
- `enter`: 変更を適用して browse mode に戻る
- `esc`: 変更を破棄して browse mode に戻る

少なくとも1つのカラムを表示する必要がある。すべてのカラムを非表示にする操作は許可しない。

browse mode のフッターとヘルプ画面に、カラム設定画面を開くキー操作を追加する。

## Column Width

各カラムの幅は、ヘッダーと現在表示する値を基に自動計算する。

カラムの合計幅が browse pane の幅を超える場合は、次の順に調整する。

1. `message` カラムを優先的に縮小する。
2. それでも収まらない場合は、ほかのカラムも縮小する。

セルの内容を幅に合わせて短縮した場合は、末尾を `…` とする。

## CLI

TUI の初期カラムを `--columns` オプションで指定できる。

```console
wwlog --file app.jsonl --tui \
  --columns timestamp,status,@http.status,message
```

- 値はカンマ区切りとし、指定順を表示順として扱う。
- `--columns` は `--tui` 専用とする。
- `--tui` なしで `--columns` を指定した場合は、`--columns requires --tui` というエラーにする。
- 不正な組み込みカラム名はエラーにする。
- 入力ファイル内に存在しない `@attribute` の指定は許可し、該当カラムのすべてのセルを空欄にする。
- `--columns` で指定した設定は TUI の初期値とし、起動後は設定画面で一時的に変更できる。
- `--columns` が指定されていない場合はデフォルトカラムを使用する。

## Out of Scope

- カラム設定の永続化
- カラム幅の手動設定
- カラムヘッダー名の変更
- timestamp の表示形式の変更
- detail pane のカスタマイズ
- 非 TUI のテーブル出力のカスタマイズ


# ccsl

`ccsl` (**c**laude **c**ode **s**tatus **l**ine) は [Claude Code の status line](https://code.claude.com/docs/en/statusline.md) を描画する Go 製コマンドです。
Claude Code が stdin に流すセッション JSON を読み、4 行のステータスラインを stdout に出力します。

```
🏷️ statusline の実装
🤖 Opus │ 📁 ~/ghq/github.com/yteraoka/ccsl │ 🌿 main✱ ↑2 │ 🌳 my-feature ← main │ 🔗 PR #1234 👀
🧠 ████░░░░░░ 43% (85.7k/200k) │ 💾 91% 42m/1h │ ⏳ 5h 24% 2h10m │ 📅 7d 91% 3d4h │ 💰 $1.23
🆔 2fa45908-49bb-4048-b74c-e58d273f075a │ ⏱️ 1h15m │ ⚡ 12m03s
```

1 行目にセッション名、2 行目に「どこで作業しているか」、3 行目に「どれだけ消費しているか」、4 行目にセッション ID と経過時間を表示します。

## 表示内容

| | 項目 | 説明 |
|---|---|---|
| 🏷️ | セッション名 | `--name` / `/rename` で付けた名前、または AI が生成したタイトル |
| 🤖 | モデル名 | `model.display_name` |
| 📁 | 作業ディレクトリ | `$HOME` は `~` に短縮。幅が足りなければ `…/末尾` に省略 |
| 🌿 | git ブランチ | `git` から取得。`✱` = 未コミットの変更、`↑`/`↓` = upstream との差分。detached HEAD は `@abc1234` |
| 🌳 | git worktree | worktree セッション名と分岐元ブランチ (`← main`)。通常の linked worktree は `workspace.git_worktree` |
| 🔗 | Pull Request | `pr.number` / `pr.url`。OSC 8 でクリック可能。レビュー状態は ✅ approved / ❌ changes\_requested / 👀 pending / 📝 draft。GitLab の場合は `MR !123` |
| 🧠 | トークン消費率 | コンテキストウィンドウ使用率のバー + % + 実トークン数。70% で黄、90% で赤。200k 超は `⚠` |
| 💾 | プロンプトキャッシュ | ヒット率 + キャッシュの残り寿命（`42m/1h` = 1 時間の TTL のうち残り 42 分）。ヒット率は 80% 以上で緑、50% 以上で黄、それ未満は赤。キャッシュが切れていれば `cold`、ミスがあれば `miss N`、キャッシュが使われていなければ `off` |
| ⏳ | 5 時間リミット | 使用率と、リセットまでの残り時間 |
| 📅 | 7 日リミット | 同上 |
| 💳 | スペンドリミット | Claude apps gateway 配下の場合のみ |
| 💰 | コスト | `cost.total_cost_usd`（クライアント側の概算） |
| ⏱️ | セッション継続時間 | `cost.total_duration_ms` |
| ⚡ | API 合計時間 | `cost.total_api_duration_ms` |
| 🆔 | セッション ID | `session_id` を省略せず全体表示 |

JSON に含まれない項目（名前が付いていない、PR がない、worktree ではない、サブスクリプションのレート制限が届いていない等）は自動的に省略されます。行の中身がすべて無い場合はその行ごと出力しません。

## インストール

[mise](https://mise.jdx.dev/) で GitHub Release のバイナリを入れる場合:

```sh
mise use -g github:yteraoka/ccsl@1.0.1
```

Go ツールチェインがある場合:

```sh
go install github.com/yteraoka/ccsl@latest
```

ソースから:

```sh
git clone https://github.com/yteraoka/ccsl.git
cd ccsl
go build -o ccsl .
```

## 設定

`~/.claude/settings.json`（またはプロジェクトの `.claude/settings.json`）に追加します。

```json
{
  "statusLine": {
    "type": "command",
    "command": "ccsl",
    "padding": 0
  }
}
```

`PATH` に入れていない場合は絶対パスを指定してください。

```json
{
  "statusLine": {
    "type": "command",
    "command": "/home/you/go/bin/ccsl"
  }
}
```

レート制限のカウントダウンやセッション時間をアイドル中も更新したい場合は `refreshInterval` を指定します。

```json
{
  "statusLine": {
    "type": "command",
    "command": "ccsl",
    "refreshInterval": 30000
  }
}
```

## オプション

| フラグ | 説明 |
|---|---|
| `--one-line` | 4 行ではなく 1 行にまとめる |
| `--no-emoji` | 絵文字の代わりにテキストラベル (`model` / `dir` / `ctx` …) を使う |
| `--no-color` | ANSI カラーを出力しない（環境変数 `NO_COLOR` でも同じ） |
| `--no-links` | PR の OSC 8 クリッカブルリンクを無効化（未対応ターミナル向け） |
| `--no-git-status` | `✱` と `↑`/`↓` の取得をやめて git 呼び出しを 1 回に減らす |
| `--bar-width N` | コンテキスト使用率バーの幅（既定 10） |
| `--version` | バージョンを表示 |

## 動作確認

Claude Code を経由せずに試すには、JSON を直接流し込みます。

```sh
echo '{
  "cwd": "'"$PWD"'",
  "session_id": "2fa45908-49bb-4048-b74c-e58d273f075a",
  "model": {"display_name": "Opus"},
  "workspace": {"current_dir": "'"$PWD"'"},
  "cost": {"total_cost_usd": 1.23, "total_duration_ms": 4500000, "total_api_duration_ms": 723000},
  "context_window": {"total_input_tokens": 84500, "total_output_tokens": 1200,
                     "context_window_size": 200000, "used_percentage": 42.85}
}' | ccsl
```

## 補足

- Claude Code はアシスタントのメッセージごとにこのコマンドを実行するため、git 呼び出しには 400ms のタイムアウトを設けています。取得できなければブランチ表示を省略するだけで、ステータスラインが止まることはありません。
- ターミナル幅は Claude Code が渡す環境変数 `COLUMNS` から読み取り、はみ出す場合はディレクトリを省略、それでも収まらなければ行末を `…` で切り詰めます。
- `rate_limits` は Claude.ai Pro / Max のサブスクリプションで、かつセッション最初の API 応答以降にのみ届きます。
- `prompt_cache` もセッション最初の API 応答以降にのみ届きます（メインの会話のみが対象で、サブエージェントの分は含まれません）。

## 開発

Go / golangci-lint / goreleaser のバージョンは [mise](https://mise.jdx.dev/) で管理しています。

```sh
mise install     # mise.toml / mise.lock のバージョンを揃える
go test ./...
go vet ./...
golangci-lint run
```

`mise.lock` に各プラットフォームのチェックサムを記録しているので、CI もローカルも同じバイナリを取得します。バージョンを上げるときは `mise.toml` を編集してから `mise lock` を実行してください。

GitHub Actions のアクションは [pinact](https://github.com/suzuki-shunsuke/pinact) でコミット SHA に固定しています。ワークフローを追加・更新したら実行してください。

```sh
pinact run
```

## CI / リリース

- **CI** (`.github/workflows/ci.yml`) — Pull Request と `main` への push で、`go build` / `go vet` / `go test -race -cover`、`golangci-lint run`、そして `goreleaser check` と全プラットフォーム向けのスナップショットビルドを実行します。
- **tagpr** (`.github/workflows/tagpr.yml`) — `main` への push で [tagpr](https://github.com/Songmu/tagpr) がリリース PR を維持します。マージするとタグが打たれます。
- **Release** (`.github/workflows/release.yml`) — `v*` のタグを push すると [goreleaser](https://goreleaser.com/) が linux / macOS / Windows の amd64・arm64 向けバイナリをビルドし、GitHub Release を作成します。

リリースの流れは次のとおりです。

1. `main` に変更が入ると tagpr が「次のバージョン」のリリース PR を作成・更新します（CHANGELOG と、上のインストール手順に書かれている mise のバージョンも更新）。
2. そのリリース PR をマージすると tagpr が `vX.Y.Z` のタグを打ちます。メジャー / マイナーを上げたい場合は、PR に `major` / `minor` ラベルを付けてからマージします。
3. タグの push で Release ワークフローが動き、goreleaser が成果物付きの GitHub Release を公開します。

タグを手動で打っても同じく Release ワークフローが動きます。

```sh
git tag v0.1.0
git push origin v0.1.0
```

### tagpr のセットアップ

タグは GitHub App のトークンで push します。`GITHUB_TOKEN` で作成したタグは別のワークフローを起動しないため、それでは Release ワークフローが動かないからです。

リポジトリに以下を設定してください。

| 種別 | 名前 | 内容 |
|---|---|---|
| Variables | `TAGPR_APP_ID` | GitHub App の Client ID (`Iv23...`) |
| Secrets | `TAGPR_APP_PRIVATE_KEY` | GitHub App の秘密鍵 |

App に必要な権限は Contents: Read and write / Pull requests: Read and write / Issues: Read-only です。

なお GitHub Release は goreleaser が作るため、`.tagpr` では `release = false` にしてあります。バージョンはタグからのみ決まり（`versionFile = -`）、ビルド時に `-X main.version` で埋め込まれます。

README のインストール手順に書かれているバージョンは、tagpr がリリース PR を作るときに `tagpr.command` から `scripts/update-readme-version` を呼んで書き換えます。手で直す必要はありません。

リリース前に成果物を確認するには:

```sh
goreleaser build --snapshot --clean
```

## License

MIT

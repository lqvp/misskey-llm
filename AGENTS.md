# AGENTS.md - Misskey LLM Bot

## プロジェクト概要

Misskey上でOpenAI互換API（Xiaomi MiMo等）を使用したLLMボットを動作させるプロジェクト。

## 技術スタック

- **言語**: Go
- **WebSocket**: gorilla/websocket
- **設定管理**: joho/godotenv
- **API**: Misskey Streaming API, OpenAI互換API

## ディレクトリ構成

- `main.go` - エントリーポイント
- `config.go` - 環境変数設定管理
- `misskey/` - Misskey APIクライアント
- `llm/` - LLM APIクライアント
- `bot/` - ボットロジック
- `storage/` - 会話履歴保存

## コーディング規約

- Go標準のコーディングスタイルに従う
- エラーハンドリングを適切に実装
- ログ出力でデバッグ情報を記録
- 環境変数で設定を管理

## ビルド・実行

```bash
go build -o misskey-llm .
./misskey-llm
```

## テスト

```bash
go test ./...
```

## 環境変数

- `MISSKEY_URL` - MisskeyインスタンスURL
- `MISSKEY_TOKEN` - ボットアクセストークン
- `LLM_ENDPOINT` - OpenAI互換APIエンドポイント
- `LLM_API_KEY` - APIキー
- `LLM_MODEL` - モデル名
- `MAX_TOKENS` - 最大トークン数
- `SYSTEM_PROMPT` - システムプロンプト

## 注意事項

- アクセストークンは`.env`ファイルで管理し、Gitにコミットしない
- WebSocket接続は自動再接続機能付き
- 会話履歴は`data/conversations/`に保存

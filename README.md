# Misskey LLM Bot

Misskey上でOpenAI互換API（Xiaomi MiMo等）を使ったLLMボット。

## セットアップ

```bash
cp .env.example .env
# .envを編集

go build -o misskey-llm .
./misskey-llm
```

## 環境変数

| 変数名 | 説明 | デフォルト |
|--------|------|-----------|
| `MISSKEY_URL` | MisskeyインスタンスのURL | 必須 |
| `MISSKEY_TOKEN` | ボット用アクセストークン | 必須 |
| `LLM_ENDPOINT` | OpenAI互換APIのエンドポイント | 必須 |
| `LLM_API_KEY` | APIキー | 必須 |
| `LLM_MODEL` | 使用するモデル名 | `mimo-v2.5-pro` |
| `MAX_TOKENS` | 応答の最大トークン数 | `1024` |
| `TEMPERATURE` | 応答の創造性 (0.0-1.5) | `1.0` |
| `TOP_P` | 核サンプリング確率 | `0.95` |
| `ENABLE_SEARCH` | MiMo Web Search | `false` |
| `ENABLE_THINKING` | 思考モード | `true` |
| `LOG_LEVEL` | ログレベル (error/info/debug) | `info` |
| `SYSTEM_PROMPT` | システムプロンプト | デフォルトあり |

## 機能

- WebSocketでMisskey通知をリアルタイム監視
- mention、reply、quoteに自動応答
- リモートユーザーへの`@id@host`メンション
- quote時に元投稿を参照
- URL自動検出・内容取得（Jina Reader）
- ツール呼び出し（Jina Search、URL Fetch）
- 会話履歴の永続化
- ファイルログ出力

## ディレクトリ構成

```
misskey-llm/
├── main.go
├── config.go
├── bot/bot.go
├── misskey/client.go, types.go
├── llm/client.go
├── jina/client.go
├── logger/logger.go
├── storage/storage.go
├── log/              # ログファイル
└── data/conversations/  # 会話履歴
```

## Misskeyアクセストークン

1. 設定 > API > トークンの生成
2. 権限: `read:notifications`, `write:notes`

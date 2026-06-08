# Misskey LLM Bot

OpenAI互換APIを使用してMisskey上でLLMボットを動作させるプロジェクト。

## 概要

MisskeyのWebSocket Streaming APIを使用して通知（mention、reply、quote）をリアルタイムで監視し、OpenAI互換API（Xiaomi MiMo等）を使用して応答を生成するボット。

## アーキテクチャ

```
┌─────────────────┐     WebSocket     ┌──────────────────┐
│   Misskey API   │ ◄───────────────► │   misskey/client │
└─────────────────┘                   └──────────────────┘
                                              │
                                              ▼
┌─────────────────┐     REST API      ┌──────────────────┐
│   LLM (MiMo等)  │ ◄───────────────► │    llm/client    │
└─────────────────┘                   └──────────────────┘
                                              │
                                              ▼
                                      ┌──────────────────┐
                                      │    bot/bot.go    │
                                      │  (通知処理・応答) │
                                      └──────────────────┘
                                              │
                                              ▼
                                      ┌──────────────────┐
                                      │ storage/storage  │
                                      │ (会話履歴保存)    │
                                      └──────────────────┘
```

## ディレクトリ構成

```
misskey-llm/
├── main.go              # エントリーポイント
├── config.go            # 環境変数による設定管理
├── .env.example         # 環境変数テンプレート
├── docs/
│   ├── README.md        # このファイル
│   └── openapi-docs.yaml # Misskey API仕様
├── misskey/
│   ├── client.go        # Misskey WebSocket/RESTクライアント
│   └── types.go         # Misskeyデータ型定義
├── llm/
│   └── client.go        # OpenAI互換APIクライアント
├── bot/
│   └── bot.go           # ボットロジック
└── storage/
    └── storage.go       # 会話履歴のJSON保存
```

## セットアップ

### 1. 環境変数の設定

`.env.example`をコピーして`.env`を作成し、必要な値を設定する。

```bash
cp .env.example .env
```

| 変数名 | 説明 | 必須 |
|--------|------|------|
| `MISSKEY_URL` | MisskeyインスタンスのURL | ✓ |
| `MISSKEY_TOKEN` | ボット用アクセストークン | ✓ |
| `LLM_ENDPOINT` | OpenAI互換APIのエンドポイント | ✓ |
| `LLM_API_KEY` | APIキー | ✓ |
| `LLM_MODEL` | 使用するモデル名 | |
| `MAX_TOKENS` | 応答の最大トークン数 | |
| `SYSTEM_PROMPT` | システムプロンプト | |

### 2. Misskeyアクセストークンの取得

1. Misskeyインスタンスにログイン
2. 設定 > API > トークンの生成
3. 必要な権限を選択:
   - `read:notifications` - 通知の読み取り
   - `write:notes` - ノートの作成

### 3. ビルドと実行

```bash
# ビルド
go build -o misskey-llm .

# 実行
./misskey-llm
```

または直接:

```bash
go run .
```

## 動作原理

1. **WebSocket接続**: MisskeyのStreaming APIに接続し、`main`チャンネルで通知を監視
2. **通知処理**: `mention`、`reply`、`quote`タイプの通知を検出
3. **コンテキスト取得**: 必要に応じて会話履歴をMisskey APIから取得
4. **LLM応答生成**: OpenAI互換APIを使用して応答を生成
5. **リプライ送信**: `notes/create` APIを使用してリプライを投稿
6. **履歴保存**: 会話履歴をJSONファイルに保存

## 会話履歴

会話履歴は`data/conversations/`ディレクトリにJSONファイルとして保存される。

- ユーザーごとに個別ファイル（`{userId}.json`）
- 直近20件のメッセージを保持
- ボット再起動後も履歴を維持

## トラブルシューティング

### WebSocket接続が切れる

自動再接続機能が実装されている。接続が切れた場合、5秒後に再接続を試行する。

### LLM APIエラー

APIエラーが発動した場合、フォールバック応答を返す。

### 権限エラー

Misskeyアクセストークンの権限を確認してください。

## ライセンス

MIT License

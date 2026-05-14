# Echo Todo API

Go + Echo + PostgreSQL で構築したシンプルなタスク管理 REST API です。
JWT 認証・レートリミット・Graceful Shutdown など、本番運用を想定した実装になっています。

---

## 目次

- [技術スタック](#技術スタック)
- [ディレクトリ構成](#ディレクトリ構成)
- [セットアップ](#セットアップ)
- [環境変数](#環境変数)
- [API リファレンス](#api-リファレンス)
- [認証フロー](#認証フロー)
- [バリデーションルール](#バリデーションルール)
- [ページネーション](#ページネーション)
- [ヘルスチェック](#ヘルスチェック)

---

## 技術スタック

| カテゴリ           | 採用技術                                          |
| ------------------ | ------------------------------------------------- |
| 言語               | Go 1.23                                           |
| Web フレームワーク | [Echo v4](https://echo.labstack.com/)             |
| 認証               | JWT (HS256) — `golang-jwt/jwt v5` + `echo-jwt/v4` |
| データベース       | PostgreSQL 15                                     |
| DB ドライバ        | `lib/pq`                                          |
| バリデーション     | `go-playground/validator v10`                     |
| コンテナ           | Docker / Docker Compose                           |

---

## ディレクトリ構成

```
echo-todo-api/
├── cmd/
│   └── server/
│       └── main.go               # エントリーポイント・DI・Graceful Shutdown
├── internal/
│   ├── config/
│   │   └── config.go             # 環境変数の読み込みと一元管理
│   ├── domain/
│   │   └── task.go               # Task モデル・リクエスト/レスポンス DTO
│   ├── handler/
│   │   ├── auth.go               # POST /auth/login（JWT 発行）
│   │   ├── health.go             # GET /health（DB 疎通確認）
│   │   └── task.go               # タスク CRUD ハンドラー
│   ├── repository/
│   │   ├── interface.go          # TaskRepository インターフェース定義
│   │   └── postgres.go           # PostgreSQL 実装
│   ├── usecase/
│   │   └── task.go               # ビジネスロジック層
│   └── validator/
│       └── validator.go          # カスタムバリデーター（日本語エラー）
├── migrations/
│   └── 001_create_tables.sql     # テーブル定義（users / tasks）
├── .env.example                  # 環境変数テンプレート
├── .gitignore
├── Dockerfile                    # マルチステージビルド
├── docker-compose.yml
└── go.mod
```

---

## セットアップ

### 前提条件

- Go 1.23 以上
- Docker / Docker Compose

### 1. リポジトリのクローンと環境変数の設定

```bash
git clone <repository-url>
cd echo-todo-api

# テンプレートをコピーして値を編集する
cp .env.example .env
```

`.env` を開き、最低限以下の 2 つを変更してください。

```env
DB_PASSWORD=your_strong_password
JWT_SECRET=your_very_long_random_secret   # openssl rand -base64 32 で生成推奨
```

### 2. データベースの起動

```bash
docker compose up -d db
```

### 3. アプリケーションの起動

**Docker で動かす場合（推奨）:**

```bash
docker compose up --build
```

**ローカルで直接動かす場合:**

```bash
go mod tidy
go run ./cmd/server
```

起動確認：

```bash
curl http://localhost:8080/health
# → {"status":"ok","checks":{"database":"ok"}}
```

---

## 環境変数

| 変数名                     | 必須 | デフォルト              | 説明                                          |
| -------------------------- | ---- | ----------------------- | --------------------------------------------- |
| `DB_PASSWORD`              | ✅   | —                       | DB パスワード                                 |
| `JWT_SECRET`               | ✅   | —                       | JWT 署名シークレット（32 文字以上推奨）       |
| `DB_HOST`                  |      | `127.0.0.1`             | DB ホスト                                     |
| `DB_PORT`                  |      | `15432`                 | DB ポート                                     |
| `DB_USER`                  |      | `postgres`              | DB ユーザー                                   |
| `DB_NAME`                  |      | `todo`                  | DB 名                                         |
| `DB_SSLMODE`               |      | `disable`               | SSL モード（本番は `require`）                |
| `DB_MAX_OPEN_CONNS`        |      | `25`                    | コネクションプール最大数                      |
| `DB_MAX_IDLE_CONNS`        |      | `10`                    | アイドル接続の最大保持数                      |
| `DB_CONN_MAX_LIFETIME_MIN` |      | `5`                     | 接続の最大生存時間（分）                      |
| `SERVER_PORT`              |      | `8080`                  | サーバーポート                                |
| `ALLOWED_ORIGINS`          |      | `http://localhost:3000` | CORS 許可オリジン（カンマ区切りで複数指定可） |
| `RATE_LIMIT`               |      | `20`                    | レートリミット（リクエスト数/秒）             |
| `JWT_EXPIRES_HOURS`        |      | `72`                    | JWT 有効期限（時間）                          |
| `SEED_USER_EMAIL`          |      | —                       | 開発用ログインメール                          |
| `SEED_USER_PASSWORD`       |      | —                       | 開発用ログインパスワード                      |

---

## API リファレンス

### エンドポイント一覧

| メソッド | パス                | 認証 | 説明                               |
| -------- | ------------------- | ---- | ---------------------------------- |
| `POST`   | `/auth/login`       | 不要 | JWT トークンの発行                 |
| `GET`    | `/health`           | 不要 | ヘルスチェック                     |
| `GET`    | `/api/v1/tasks`     | 必要 | タスク一覧取得（ページネーション） |
| `POST`   | `/api/v1/tasks`     | 必要 | タスク作成                         |
| `GET`    | `/api/v1/tasks/:id` | 必要 | タスク取得                         |
| `PUT`    | `/api/v1/tasks/:id` | 必要 | タスク更新                         |
| `DELETE` | `/api/v1/tasks/:id` | 必要 | タスク削除（ソフトデリート）       |

---

### 認証 — ログイン

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@example.com", "password": "changeme"}'
```

**レスポンス:**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 259200
}
```

取得したトークンは以降のリクエストで `Authorization: Bearer <token>` ヘッダーに付与します。

---

### タスク一覧取得

```bash
curl http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer <token>"

# ページ・件数を指定する場合
curl "http://localhost:8080/api/v1/tasks?page=2&limit=5" \
  -H "Authorization: Bearer <token>"
```

**レスポンス:**

```json
{
  "data": [
    {
      "id": 1,
      "title": "買い物をする",
      "completed": false,
      "created_at": "2025-01-01T10:00:00+09:00",
      "updated_at": "2025-01-01T10:00:00+09:00"
    }
  ],
  "total": 25,
  "page": 2,
  "limit": 5,
  "total_pages": 5
}
```

---

### タスク作成

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"title": "買い物をする", "completed": false}'
```

**レスポンス `201 Created`:**

```json
{
  "id": 1,
  "title": "買い物をする",
  "completed": false,
  "created_at": "2025-01-01T10:00:00+09:00",
  "updated_at": "2025-01-01T10:00:00+09:00"
}
```

---

### タスク取得・更新・削除

```bash
# 特定タスクを取得
curl http://localhost:8080/api/v1/tasks/1 \
  -H "Authorization: Bearer <token>"

# タスクを更新
curl -X PUT http://localhost:8080/api/v1/tasks/1 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"title": "買い物をする（完了）", "completed": true}'

# タスクを削除（ソフトデリート）
curl -X DELETE http://localhost:8080/api/v1/tasks/1 \
  -H "Authorization: Bearer <token>"
```

---

### エラーレスポンス

全エラーは統一フォーマットで返します。

```json
{ "error": "タスクが見つかりません" }
```

バリデーションエラー（`422 Unprocessable Entity`）はフィールドごとの詳細を含みます。

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"title": ""}'
```

```json
{
  "message": "バリデーションエラーが発生しました",
  "errors": {
    "Title": "必須項目です"
  }
}
```

| HTTP ステータス             | 意味                           |
| --------------------------- | ------------------------------ |
| `200 OK`                    | 成功                           |
| `201 Created`               | 作成成功                       |
| `204 No Content`            | 削除成功                       |
| `400 Bad Request`           | リクエスト形式が不正           |
| `401 Unauthorized`          | 認証失敗またはトークン期限切れ |
| `404 Not Found`             | リソースが見つからない         |
| `422 Unprocessable Entity`  | バリデーションエラー           |
| `429 Too Many Requests`     | レートリミット超過             |
| `500 Internal Server Error` | サーバー内部エラー             |
| `503 Service Unavailable`   | DB 接続不可                    |

---

## バリデーションルール

| フィールド | ルール            |
| ---------- | ----------------- |
| `title`    | 必須、1〜100 文字 |

---

## ページネーション

| パラメータ | デフォルト | 上限  | 説明                 |
| ---------- | ---------- | ----- | -------------------- |
| `page`     | `1`        | —     | ページ番号（1 以上） |
| `limit`    | `10`       | `100` | 1 ページあたりの件数 |

---

## ヘルスチェック

```bash
curl http://localhost:8080/health
```

**正常時 `200 OK`:**

```json
{
  "status": "ok",
  "checks": {
    "database": "ok"
  }
}
```

**異常時 `503 Service Unavailable`:**

```json
{
  "status": "ng",
  "checks": {
    "database": "ng: dial tcp: connection refused"
  }
}
```

Kubernetes の Liveness / Readiness Probe や Docker Compose の `healthcheck` にそのまま利用できます。

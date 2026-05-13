## 実行方法

### 1. 必要なライブラリのインストール

```bash
go mod tidy
```

### 2. 実行

```bash
go run main.go
```

---

## APIの確認

### タスク一覧の取得（ページネーション対応）

```bash
# デフォルト（1ページ目、10件）
curl http://localhost:8080/tasks

# ページ・件数を指定
curl "http://localhost:8080/tasks?page=2&limit=5"
```

**レスポンス例:**

```json
{
  "data": [...],
  "total": 25,
  "page": 2,
  "limit": 5,
  "total_pages": 5
}
```

---

### タスクの作成

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "タスク1", "completed": false}'
```

**バリデーションエラー例（タイトルなし）:**

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "", "completed": false}'
```

```json
{
  "message": "バリデーションエラーが発生しました",
  "errors": {
    "Title": "必須項目です"
  }
}
```

---

### タスクの取得・更新・削除

```bash
# 特定のタスクを取得
curl http://localhost:8080/tasks/1

# タスクの更新
curl -X PUT http://localhost:8080/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{"title": "タスク1（更新）", "completed": true}'

# タスクの削除
curl -X DELETE http://localhost:8080/tasks/1
```

---

## バリデーションルール

| フィールド | ルール           |
| ---------- | ---------------- |
| title      | 必須、1〜100文字 |

## ページネーションパラメータ

| パラメータ | デフォルト | 上限 |
| ---------- | ---------- | ---- |
| page       | 1          | -    |
| limit      | 10         | 100  |

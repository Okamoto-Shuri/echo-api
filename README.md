README.md

## 実行方法

### 1. プロジェクトの初期化

```bash
go mod init echo-todo-api
```

### 2. 必要なライブラリのインストール

```bash
go get github.com/labstack/echo/v4
```

### 3. 実行

```bash
go run main.go
```

### 4. APIの確認

```bash
# タスク一覧の取得
curl http://localhost:8080/tasks

# タスクの作成
curl -X POST http://localhost:8080/tasks -H "Content-Type: application/json" -d '{"title": "タスク1", "completed": false}'

# タスクの更新
curl -X PUT http://localhost:8080/tasks/1 -H "Content-Type: application/json" -d '{"title": "タスク1（更新）", "completed": true}'

# タスクの削除
curl -X DELETE http://localhost:8080/tasks/1
```

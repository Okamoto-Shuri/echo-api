package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	_ "github.com/lib/pq" // PostgreSQLドライバ
)

// -----------------------------------------------------------------------
// モデル
// -----------------------------------------------------------------------

type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"     validate:"required,min=1,max=100"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateTaskRequest struct {
	Title     string `json:"title"     validate:"required,min=1,max=100"`
	Completed bool   `json:"completed"`
}

type UpdateTaskRequest struct {
	Title     string `json:"title"     validate:"required,min=1,max=100"`
	Completed bool   `json:"completed"`
}

type PaginatedResponse struct {
	Data       []Task `json:"data"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	TotalPages int    `json:"total_pages"`
}

// -----------------------------------------------------------------------
// カスタムバリデーター
// -----------------------------------------------------------------------

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func validationError(err error) map[string]interface{} {
	errors := map[string]string{}
	for _, e := range err.(validator.ValidationErrors) {
		switch e.Tag() {
		case "required":
			errors[e.Field()] = "必須項目です"
		case "min":
			errors[e.Field()] = "最小" + e.Param() + "文字以上で入力してください"
		case "max":
			errors[e.Field()] = "最大" + e.Param() + "文字以内で入力してください"
		default:
			errors[e.Field()] = "入力値が不正です（" + e.Tag() + "）"
		}
	}
	return map[string]interface{}{
		"message": "バリデーションエラーが発生しました",
		"errors":  errors,
	}
}

// -----------------------------------------------------------------------
// ハンドラー（DB依存を注入する構造体）
// -----------------------------------------------------------------------

// TaskHandler はDB接続を保持し、各エンドポイントの処理を担当します。
type TaskHandler struct {
	DB *sql.DB
}

// 1. 【Read】タスク一覧を取得（ページネーション対応）
func (h *TaskHandler) getTasks(c echo.Context) error {
	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// 全件数の取得
	var total int
	err = h.DB.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&total)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "データベースエラー"})
	}

	totalPages := (total + limit - 1) / limit
	offset := (page - 1) * limit

	// レコードの取得 (OFFSET と LIMIT を使用)
	rows, err := h.DB.Query("SELECT id, title, completed, created_at FROM tasks ORDER BY id LIMIT $1 OFFSET $2", limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "データベースエラー"})
	}
	defer rows.Close()

	tasks := []Task{} // nilスライスではなく空スライスで初期化し、JSONでnullにならないようにする
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "データ読み込みエラー"})
		}
		tasks = append(tasks, t)
	}

	return c.JSON(http.StatusOK, PaginatedResponse{
		Data:       tasks,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}

// 2. 【Create】タスクを新規作成
func (h *TaskHandler) createTask(c echo.Context) error {
	req := new(CreateTaskRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "リクエストの形式が正しくありません"})
	}
	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, validationError(err))
	}

	var task Task
	// RETURNING句を使って、DB側で採番されたIDとタイムスタンプを即座に取得
	query := `INSERT INTO tasks (title, completed) VALUES ($1, $2) RETURNING id, title, completed, created_at`
	err := h.DB.QueryRow(query, req.Title, req.Completed).Scan(&task.ID, &task.Title, &task.Completed, &task.CreatedAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "タスクの保存に失敗しました"})
	}

	return c.JSON(http.StatusCreated, task)
}

// 3. 【Read】指定したIDのタスクを取得
func (h *TaskHandler) getTask(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "IDは数値で指定してください"})
	}

	var t Task
	err = h.DB.QueryRow("SELECT id, title, completed, created_at FROM tasks WHERE id = $1", id).
		Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt)

	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "タスクが見つかりません"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "データベースエラー"})
	}

	return c.JSON(http.StatusOK, t)
}

// 4. 【Update】指定したIDのタスクを更新
func (h *TaskHandler) updateTask(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "IDは数値で指定してください"})
	}

	req := new(UpdateTaskRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "リクエストの形式が正しくありません"})
	}
	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, validationError(err))
	}

	// 更新処理。対象が存在するかどうかを RowsAffected で確認
	res, err := h.DB.Exec("UPDATE tasks SET title = $1, completed = $2 WHERE id = $3", req.Title, req.Completed, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "更新に失敗しました"})
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "タスクが見つかりません"})
	}

	// 更新後のデータを取得して返す
	return h.getTask(c)
}

// 5. 【Delete】指定したIDのタスクを削除
func (h *TaskHandler) deleteTask(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "IDは数値で指定してください"})
	}

	res, err := h.DB.Exec("DELETE FROM tasks WHERE id = $1", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "削除に失敗しました"})
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "タスクが見つかりません"})
	}

	return c.NoContent(http.StatusNoContent)
}

// -----------------------------------------------------------------------
// エントリーポイント
// -----------------------------------------------------------------------

func main() {
	// データベース接続
	// ※実際の運用では環境変数(os.Getenv)から取得するようにします
	connStr := "host=127.0.0.1 port=15432 user=postgres password=postgres dbname=todo sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("データベース接続エラー: %v", err)
	}
	defer db.Close()

	// 接続確認
	if err := db.Ping(); err != nil {
		log.Fatalf("データベースPingエラー: %v", err)
	}

	// ハンドラーの初期化（DBを注入）
	h := &TaskHandler{DB: db}

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Validator = &CustomValidator{validator: validator.New()}

	// ルーティング（メソッドを紐付け）
	e.GET("/tasks", h.getTasks)
	e.POST("/tasks", h.createTask)
	e.GET("/tasks/:id", h.getTask)
	e.PUT("/tasks/:id", h.updateTask)
	e.DELETE("/tasks/:id", h.deleteTask)

	e.Logger.Fatal(e.Start(":8080"))
}

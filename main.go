package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// -----------------------------------------------------------------------
// モデル
// -----------------------------------------------------------------------

// Task はタスク1件を表す構造体です。
// validateタグでバリデーションルールを定義しています。
type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"     validate:"required,min=1,max=100"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateTaskRequest はタスク作成時のリクエストボディです。
type CreateTaskRequest struct {
	Title     string `json:"title"     validate:"required,min=1,max=100"`
	Completed bool   `json:"completed"`
}

// UpdateTaskRequest はタスク更新時のリクエストボディです。
type UpdateTaskRequest struct {
	Title     string `json:"title"     validate:"required,min=1,max=100"`
	Completed bool   `json:"completed"`
}

// -----------------------------------------------------------------------
// ページネーション用レスポンス
// -----------------------------------------------------------------------

// PaginatedResponse はページネーション情報とデータを含むレスポンス構造体です。
type PaginatedResponse struct {
	Data       []Task `json:"data"`        // 現在のページのタスク一覧
	Total      int    `json:"total"`       // 全タスク数
	Page       int    `json:"page"`        // 現在のページ番号
	Limit      int    `json:"limit"`       // 1ページあたりの件数
	TotalPages int    `json:"total_pages"` // 総ページ数
}

// -----------------------------------------------------------------------
// カスタムバリデーター（Echoへの組み込み）
// -----------------------------------------------------------------------

// CustomValidator はgo-playground/validatorをEchoに組み込むためのアダプターです。
type CustomValidator struct {
	validator *validator.Validate
}

// Validate はEchoのValidatorインターフェースを満たすメソッドです。
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// validationError はバリデーションエラーを人間が読みやすい形式に変換します。
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
// インメモリストレージ
// -----------------------------------------------------------------------

var tasks = []Task{}
var nextID = 1

// -----------------------------------------------------------------------
// ハンドラー
// -----------------------------------------------------------------------

// 1. 【Read】タスク一覧を取得（ページネーション対応）
//
//	GET /tasks?page=1&limit=10
//	  page  : ページ番号（デフォルト: 1）
//	  limit : 1ページあたりの件数（デフォルト: 10、最大: 100）
func getTasks(c echo.Context) error {
	// --- クエリパラメータのパースとデフォルト値の設定 ---
	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100 // 上限を設けて過大なリクエストを防ぐ
	}

	// --- ページネーション計算 ---
	total := len(tasks)
	totalPages := (total + limit - 1) / limit // 切り上げ除算

	// ページ番号が範囲外の場合は空のデータを返す
	start := (page - 1) * limit
	if start >= total {
		return c.JSON(http.StatusOK, PaginatedResponse{
			Data:       []Task{},
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		})
	}

	// スライスの終端インデックスが配列長を超えないよう調整
	end := start + limit
	if end > total {
		end = total
	}

	return c.JSON(http.StatusOK, PaginatedResponse{
		Data:       tasks[start:end],
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}

// 2. 【Create】タスクを新規作成
//
//	POST /tasks
func createTask(c echo.Context) error {
	req := new(CreateTaskRequest)

	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "リクエストの形式が正しくありません",
		})
	}

	// バリデーション実行
	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, validationError(err))
	}

	task := Task{
		ID:        nextID,
		Title:     req.Title,
		Completed: req.Completed,
		CreatedAt: time.Now(),
	}
	nextID++
	tasks = append(tasks, task)

	return c.JSON(http.StatusCreated, task)
}

// 3. 【Read】指定したIDのタスクを取得
//
//	GET /tasks/:id
func getTask(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "IDは数値で指定してください",
		})
	}

	for _, t := range tasks {
		if t.ID == id {
			return c.JSON(http.StatusOK, t)
		}
	}
	return c.JSON(http.StatusNotFound, map[string]string{
		"message": "タスクが見つかりません",
	})
}

// 4. 【Update】指定したIDのタスクを更新
//
//	PUT /tasks/:id
func updateTask(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "IDは数値で指定してください",
		})
	}

	req := new(UpdateTaskRequest)

	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "リクエストの形式が正しくありません",
		})
	}

	// バリデーション実行
	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, validationError(err))
	}

	for i, t := range tasks {
		if t.ID == id {
			tasks[i].Title = req.Title
			tasks[i].Completed = req.Completed
			return c.JSON(http.StatusOK, tasks[i])
		}
	}
	return c.JSON(http.StatusNotFound, map[string]string{
		"message": "タスクが見つかりません",
	})
}

// 5. 【Delete】指定したIDのタスクを削除
//
//	DELETE /tasks/:id
func deleteTask(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "IDは数値で指定してください",
		})
	}

	for i, t := range tasks {
		if t.ID == id {
			tasks = append(tasks[:i], tasks[i+1:]...)
			return c.NoContent(http.StatusNoContent)
		}
	}
	return c.JSON(http.StatusNotFound, map[string]string{
		"message": "タスクが見つかりません",
	})
}

// -----------------------------------------------------------------------
// エントリーポイント
// -----------------------------------------------------------------------

func main() {
	e := echo.New()

	// ミドルウェア
	e.Use(middleware.Logger())  // リクエストログ
	e.Use(middleware.Recover()) // パニック時のリカバリ

	// カスタムバリデーターの登録
	e.Validator = &CustomValidator{validator: validator.New()}

	// ルーティング
	e.GET("/tasks", getTasks)
	e.POST("/tasks", createTask)
	e.GET("/tasks/:id", getTask)
	e.PUT("/tasks/:id", updateTask)
	e.DELETE("/tasks/:id", deleteTask)

	e.Logger.Fatal(e.Start(":8080"))
}

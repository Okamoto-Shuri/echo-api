// internal/handler/task.go
// 改善②: 全ハンドラーで c.Request().Context() を伝播させる
// 改善⑧: updateTask の二重クエリ問題を Usecase 経由の RETURNING で解決済み
package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"echo-todo-api/internal/domain"
	"echo-todo-api/internal/usecase"

	"github.com/labstack/echo/v4"
)

// TaskHandler はタスク関連の HTTP ハンドラーを保持する
type TaskHandler struct {
	uc *usecase.TaskUsecase
}

// NewTaskHandler は TaskHandler を生成する
func NewTaskHandler(uc *usecase.TaskUsecase) *TaskHandler {
	return &TaskHandler{uc: uc}
}

// GetTasks GET /api/v1/tasks
func (h *TaskHandler) GetTasks(c echo.Context) error {
	page := parsePositiveInt(c.QueryParam("page"), 1)
	limit := clamp(parsePositiveInt(c.QueryParam("limit"), 10), 1, 100)

	resp, err := h.uc.GetTasks(c.Request().Context(), page, limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "データベースエラー")
	}
	return c.JSON(http.StatusOK, resp)
}

// GetTask GET /api/v1/tasks/:id
func (h *TaskHandler) GetTask(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "IDは数値で指定してください")
	}

	task, err := h.uc.GetTask(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "データベースエラー")
	}
	if task == nil {
		return echo.NewHTTPError(http.StatusNotFound, "タスクが見つかりません")
	}
	return c.JSON(http.StatusOK, task)
}

// CreateTask POST /api/v1/tasks
func (h *TaskHandler) CreateTask(c echo.Context) error {
	req := new(domain.CreateTaskRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "リクエストの形式が正しくありません")
	}
	if err := c.Validate(req); err != nil {
		return err // カスタムバリデーターが 422 HTTPError を返す
	}

	task, err := h.uc.CreateTask(c.Request().Context(), req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "タスクの保存に失敗しました")
	}
	return c.JSON(http.StatusCreated, task)
}

// UpdateTask PUT /api/v1/tasks/:id
func (h *TaskHandler) UpdateTask(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "IDは数値で指定してください")
	}

	req := new(domain.UpdateTaskRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "リクエストの形式が正しくありません")
	}
	if err := c.Validate(req); err != nil {
		return err
	}

	// 改善⑧: RETURNING 句で一発取得。旧実装の h.getTask(c) 再呼び出しを廃止
	task, err := h.uc.UpdateTask(c.Request().Context(), id, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "更新に失敗しました")
	}
	if task == nil {
		return echo.NewHTTPError(http.StatusNotFound, "タスクが見つかりません")
	}
	return c.JSON(http.StatusOK, task)
}

// DeleteTask DELETE /api/v1/tasks/:id
func (h *TaskHandler) DeleteTask(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "IDは数値で指定してください")
	}

	err = h.uc.DeleteTask(c.Request().Context(), id)
	if err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "タスクが見つかりません")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "削除に失敗しました")
	}
	return c.NoContent(http.StatusNoContent)
}

// ---- ヘルパー ----

func parsePositiveInt(s string, defaultVal int) int {
	if v, err := strconv.Atoi(s); err == nil && v > 0 {
		return v
	}
	return defaultVal
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

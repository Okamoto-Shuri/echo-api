// internal/domain/task.go
// 改善⑤: updated_at フィールドを Task に追加
package domain

import "time"

// Task はデータベースの tasks テーブルに対応するモデル
type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// deleted_at は外部に公開しないため JSON タグなし
}

// CreateTaskRequest は POST /tasks のリクエストボディ
type CreateTaskRequest struct {
	Title     string `json:"title"     validate:"required,min=1,max=100"`
	Completed bool   `json:"completed"`
}

// UpdateTaskRequest は PUT /tasks/:id のリクエストボディ
type UpdateTaskRequest struct {
	Title     string `json:"title"     validate:"required,min=1,max=100"`
	Completed bool   `json:"completed"`
}

// PaginatedResponse はページネーション付きタスク一覧レスポンス
type PaginatedResponse struct {
	Data       []Task `json:"data"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	TotalPages int    `json:"total_pages"`
}

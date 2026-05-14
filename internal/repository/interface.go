// internal/repository/interface.go
// インターフェースを分離することでモックに差し替え可能にし、テストを容易にする
package repository

import (
	"context"

	"echo-todo-api/internal/domain"
)

// TaskRepository はタスクのデータアクセスを抽象化するインターフェース
type TaskRepository interface {
	FindAll(ctx context.Context, page, limit int) ([]domain.Task, int, error)
	FindByID(ctx context.Context, id int) (*domain.Task, error)
	Create(ctx context.Context, req *domain.CreateTaskRequest) (*domain.Task, error)
	Update(ctx context.Context, id int, req *domain.UpdateTaskRequest) (*domain.Task, error)
	Delete(ctx context.Context, id int) error
}

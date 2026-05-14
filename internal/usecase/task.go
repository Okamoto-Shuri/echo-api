// internal/usecase/task.go
// Handler と Repository の間に立つビジネスロジック層。
// 現在はシンプルだが、将来的にトランザクション処理・通知・キャッシュなどを
// ここに追加することで Handler を汚染しない設計にする。
package usecase

import (
	"context"

	"echo-todo-api/internal/domain"
	"echo-todo-api/internal/repository"
)

// TaskUsecase はタスクに関するユースケースを実装する
type TaskUsecase struct {
	repo repository.TaskRepository
}

// NewTaskUsecase は TaskUsecase を生成する（依存性注入）
func NewTaskUsecase(repo repository.TaskRepository) *TaskUsecase {
	return &TaskUsecase{repo: repo}
}

// GetTasks はページネーション付きのタスク一覧を返す
func (u *TaskUsecase) GetTasks(ctx context.Context, page, limit int) (*domain.PaginatedResponse, error) {
	tasks, total, err := u.repo.FindAll(ctx, page, limit)
	if err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return &domain.PaginatedResponse{
		Data:       tasks,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// GetTask は指定 ID のタスクを返す。nil は「見つからない」を意味する
func (u *TaskUsecase) GetTask(ctx context.Context, id int) (*domain.Task, error) {
	return u.repo.FindByID(ctx, id)
}

// CreateTask は新しいタスクを作成する
func (u *TaskUsecase) CreateTask(ctx context.Context, req *domain.CreateTaskRequest) (*domain.Task, error) {
	return u.repo.Create(ctx, req)
}

// UpdateTask はタスクを更新する。nil は「見つからない」を意味する
func (u *TaskUsecase) UpdateTask(ctx context.Context, id int, req *domain.UpdateTaskRequest) (*domain.Task, error) {
	return u.repo.Update(ctx, id, req)
}

// DeleteTask はタスクをソフトデリートする
func (u *TaskUsecase) DeleteTask(ctx context.Context, id int) error {
	return u.repo.Delete(ctx, id)
}

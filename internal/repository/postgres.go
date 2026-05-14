// internal/repository/postgres.go
// 改善②: 全クエリに context.Context を付与してタイムアウト制御を可能にする
// 改善⑧: UpdateTask を RETURNING 句で実装し、二重クエリを排除する
// 改善⑤: ソフトデリート（deleted_at）を実装する
package repository

import (
	"context"
	"database/sql"

	"echo-todo-api/internal/domain"
)

type taskPostgresRepo struct {
	db *sql.DB
}

// NewTaskPostgresRepo は TaskRepository の PostgreSQL 実装を返す
func NewTaskPostgresRepo(db *sql.DB) TaskRepository {
	return &taskPostgresRepo{db: db}
}

// FindAll は deleted_at IS NULL のタスクをページネーションで返す
func (r *taskPostgresRepo) FindAll(ctx context.Context, page, limit int) ([]domain.Task, int, error) {
	// 改善②: QueryRowContext でリクエスト Context を伝播させる
	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM tasks WHERE deleted_at IS NULL`,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, title, completed, created_at, updated_at
		   FROM tasks
		  WHERE deleted_at IS NULL
		  ORDER BY id
		  LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tasks := []domain.Task{} // nil スライス回避（JSON で [] になる）
	for rows.Next() {
		var t domain.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

// FindByID は指定 ID のアクティブなタスクを返す。存在しない場合は (nil, nil) を返す
func (r *taskPostgresRepo) FindByID(ctx context.Context, id int) (*domain.Task, error) {
	var t domain.Task
	err := r.db.QueryRowContext(ctx,
		`SELECT id, title, completed, created_at, updated_at
		   FROM tasks
		  WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt, &t.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil // 見つからない = エラーではなく nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Create は新しいタスクを作成し、RETURNING 句で挿入結果を即時取得する
func (r *taskPostgresRepo) Create(ctx context.Context, req *domain.CreateTaskRequest) (*domain.Task, error) {
	var t domain.Task
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO tasks (title, completed)
		      VALUES ($1, $2)
		 RETURNING id, title, completed, created_at, updated_at`,
		req.Title, req.Completed,
	).Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Update はタスクを更新し、RETURNING 句で更新後の値を取得する。
// 改善⑧: 以前の「更新→再取得」の二重クエリを RETURNING 一発に置き換え
func (r *taskPostgresRepo) Update(ctx context.Context, id int, req *domain.UpdateTaskRequest) (*domain.Task, error) {
	var t domain.Task
	err := r.db.QueryRowContext(ctx,
		`UPDATE tasks
		    SET title = $1,
		        completed = $2,
		        updated_at = NOW()   -- 改善⑤: 更新日時を明示的に更新
		  WHERE id = $3 AND deleted_at IS NULL
		RETURNING id, title, completed, created_at, updated_at`,
		req.Title, req.Completed, id,
	).Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt, &t.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Delete はソフトデリートを実行する（deleted_at に現在時刻をセット）
// 改善⑤: 物理削除ではなくソフトデリートで履歴を保持する
func (r *taskPostgresRepo) Delete(ctx context.Context, id int) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows // 呼び出し元で 404 に変換する
	}
	return nil
}

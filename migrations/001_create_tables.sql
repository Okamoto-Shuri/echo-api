-- ============================================================
-- 001_create_tables.sql
-- 改善⑤: updated_at（更新日時）と deleted_at（ソフトデリート）を追加
-- ============================================================

CREATE TABLE IF NOT EXISTS users (
    id         SERIAL PRIMARY KEY,
    email      VARCHAR(255) NOT NULL UNIQUE,
    -- 実運用では bcrypt ハッシュを格納する
    password   VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tasks (
    id         SERIAL PRIMARY KEY,
    title      VARCHAR(100) NOT NULL,
    completed  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    -- 改善⑤: 更新日時カラム追加（UPDATE時に明示的に更新）
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    -- 改善⑤: ソフトデリート用カラム（NULL = 生存, 値あり = 削除済み）
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- ソフトデリート済みを除外する検索を高速化するための部分インデックス
CREATE INDEX idx_tasks_active ON tasks (id) WHERE deleted_at IS NULL;
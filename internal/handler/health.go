// internal/handler/health.go
// 改善⑨: ヘルスチェックエンドポイント
// コンテナオーケストレーター（k8s / ECS）の Liveness / Readiness Probe に対応する
package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// HealthHandler はサービスの稼働状況を返すハンドラー
type HealthHandler struct {
	db *sql.DB
}

// NewHealthHandler は HealthHandler を生成する
func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

type healthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// Check GET /health
// DB への疎通確認を含めた総合的なヘルスチェックを実施する。
// 正常: 200 OK / 異常: 503 Service Unavailable
func (h *HealthHandler) Check(c echo.Context) error {
	checks := map[string]string{}
	allOK := true

	// DB チェック: 2 秒でタイムアウト
	dbCtx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
	defer cancel()

	if err := h.db.PingContext(dbCtx); err != nil {
		checks["database"] = "ng: " + err.Error()
		allOK = false
	} else {
		checks["database"] = "ok"
	}

	resp := healthResponse{Checks: checks}
	if allOK {
		resp.Status = "ok"
		return c.JSON(http.StatusOK, resp)
	}

	resp.Status = "ng"
	return c.JSON(http.StatusServiceUnavailable, resp)
}

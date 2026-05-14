// cmd/server/main.go
// 改善①〜⑩を全て統合したエントリーポイント
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"echo-todo-api/internal/config"
	"echo-todo-api/internal/handler"
	"echo-todo-api/internal/repository"
	"echo-todo-api/internal/usecase"
	customvalidator "echo-todo-api/internal/validator"

	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

func main() {
	// ---- 構造化ログの初期化（Go 1.21 標準の slog） ----
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// ---- 改善①: 設定を環境変数から読み込む ----
	cfg, err := config.Load()
	if err != nil {
		slog.Error("設定の読み込みに失敗しました", "error", err)
		os.Exit(1)
	}

	// ---- DB 接続 ----
	db, err := sql.Open("postgres", cfg.DB.DSN())
	if err != nil {
		slog.Error("DB 接続オブジェクトの作成に失敗しました", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 改善③: コネクションプールを適切に設定する
	db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	db.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.DB.ConnMaxLifetime)

	// 改善②: Ping にも Context を使い、起動時タイムアウトを設ける
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := db.PingContext(pingCtx); err != nil {
		slog.Error("DB への疎通確認に失敗しました", "error", err)
		os.Exit(1)
	}
	slog.Info("DB に接続しました")

	// ---- 依存性注入（DI） ----
	taskRepo := repository.NewTaskPostgresRepo(db)
	taskUC := usecase.NewTaskUsecase(taskRepo)
	taskHandler := handler.NewTaskHandler(taskUC)
	healthHandler := handler.NewHealthHandler(db)
	authHandler := handler.NewAuthHandler(cfg.JWT.Secret, cfg.JWT.ExpiresHours)

	// ---- Echo の初期化 ----
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// ---- グローバルミドルウェア ----

	// 改善⑩: CORS — 許可オリジンを環境変数で制御
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: cfg.Server.AllowedOrigins,
		AllowMethods: []string{
			http.MethodGet, http.MethodPost,
			http.MethodPut, http.MethodDelete, http.MethodOptions,
		},
		AllowHeaders: []string{
			echo.HeaderContentType, echo.HeaderAuthorization,
		},
	}))

	// 改善⑦: レートリミット（メモリストア、分散環境では Redis に切り替え）
	e.Use(middleware.RateLimiter(
		middleware.NewRateLimiterMemoryStore(cfg.Server.RateLimit),
	))

	// 構造化ログ（slog 経由で JSON 出力）
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		LogLatency:  true,
		LogError:    true,
		HandleError: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			attrs := []any{
				"method", v.Method,
				"uri", v.URI,
				"status", v.Status,
				"latency", v.Latency,
			}
			if v.Error != nil {
				slog.ErrorContext(c.Request().Context(), "http", append(attrs, "error", v.Error)...)
			} else {
				slog.InfoContext(c.Request().Context(), "http", attrs...)
			}
			return nil
		},
	}))

	// パニックからの安全な復帰
	e.Use(middleware.Recover())

	// カスタムバリデーター
	e.Validator = customvalidator.New()

	// エラーレスポンスの統一フォーマット
	e.HTTPErrorHandler = unifiedErrorHandler

	// ---- ルーティング ----

	// 改善⑨: ヘルスチェックは認証なし（k8s の liveness/readiness probe 用）
	e.GET("/health", healthHandler.Check)

	// 認証エンドポイント（JWT 不要）
	e.POST("/auth/login", authHandler.Login)

	// 改善⑥: タスク API は JWT 必須 + API バージョニング
	api := e.Group("/api/v1")
	api.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey: cfg.JWT.Secret,
		// トークン検証失敗時のカスタムエラー
		ErrorHandler: func(c echo.Context, err error) error {
			return echo.NewHTTPError(http.StatusUnauthorized, "認証トークンが無効または期限切れです")
		},
	}))
	api.GET("/tasks", taskHandler.GetTasks)
	api.POST("/tasks", taskHandler.CreateTask)
	api.GET("/tasks/:id", taskHandler.GetTask)
	api.PUT("/tasks/:id", taskHandler.UpdateTask)
	api.DELETE("/tasks/:id", taskHandler.DeleteTask)

	// ---- 改善④: Graceful Shutdown ----
	// サーバーをゴルーチンで起動し、シグナルを受け取ったら安全に停止する
	go func() {
		slog.Info("サーバーを起動します", "port", cfg.Server.Port)
		if err := e.Start(":" + cfg.Server.Port); err != nil && err != http.ErrServerClosed {
			slog.Error("サーバーの起動に失敗しました", "error", err)
			os.Exit(1)
		}
	}()

	// OS シグナル（Ctrl+C / SIGTERM）を待機
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit
	slog.Info("シャットダウンシグナルを受信しました", "signal", sig)

	// 処理中のリクエストが完了するまで最大 10 秒待つ
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		slog.Error("Graceful Shutdown に失敗しました", "error", err)
		os.Exit(1)
	}
	slog.Info("サーバーを正常に停止しました")
}

// unifiedErrorHandler は全エラーレスポンスを統一フォーマット { "error": "..." } で返す
func unifiedErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := http.StatusInternalServerError
	var payload interface{} = "内部サーバーエラー"

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		payload = he.Message
	}

	// バリデーションエラーなど構造化メッセージはそのまま通す
	var body interface{}
	switch v := payload.(type) {
	case map[string]interface{}:
		body = v // バリデーションエラー等の構造化レスポンス
	default:
		body = map[string]interface{}{"error": payload}
	}

	if jsonErr := c.JSON(code, body); jsonErr != nil {
		slog.Error("エラーレスポンスの送信に失敗しました", "error", jsonErr)
	}
}

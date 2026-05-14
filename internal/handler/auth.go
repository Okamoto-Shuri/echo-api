// internal/handler/auth.go
// 改善⑥: JWT による認証エンドポイント
// POST /auth/login でトークンを発行する
package handler

import (
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// AuthHandler は認証関連のハンドラーを保持する
type AuthHandler struct {
	jwtSecret    []byte
	expiresHours int
	// 本番環境では DB からユーザーを取得するが、ここでは env の簡易実装
	adminEmail    string
	adminPassword string
}

// LoginRequest はログインリクエストのボディ
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// JWTClaims は JWT のペイロード定義
type JWTClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// NewAuthHandler は AuthHandler を生成する
func NewAuthHandler(jwtSecret []byte, expiresHours int) *AuthHandler {
	return &AuthHandler{
		jwtSecret:    jwtSecret,
		expiresHours: expiresHours,
		// 改善①: 認証情報も環境変数から取得
		adminEmail:    os.Getenv("SEED_USER_EMAIL"),
		adminPassword: os.Getenv("SEED_USER_PASSWORD"),
	}
}

// Login POST /auth/login
// メール・パスワードを検証し、JWT アクセストークンを返す
func (h *AuthHandler) Login(c echo.Context) error {
	req := new(LoginRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "リクエストの形式が正しくありません")
	}
	if err := c.Validate(req); err != nil {
		return err
	}

	// NOTE: 実際の本番実装では DB からユーザーを取得し bcrypt で照合する
	if req.Email != h.adminEmail || req.Password != h.adminPassword {
		// 存在確認の手がかりを与えないため email/password 両方に対して同じエラーを返す
		return echo.NewHTTPError(http.StatusUnauthorized, "メールアドレスまたはパスワードが正しくありません")
	}

	// JWT トークンの生成
	claims := &JWTClaims{
		Email: req.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(h.expiresHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(h.jwtSecret)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "トークンの生成に失敗しました")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"token":      signed,
		"expires_in": h.expiresHours * 3600, // 秒
	})
}

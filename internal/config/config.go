// internal/config/config.go
// 改善①: 全設定を環境変数から読み込み、ハードコードを完全排除する
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config はアプリケーション全体の設定を保持する
type Config struct {
	Server ServerConfig
	DB     DBConfig
	JWT    JWTConfig
}

type ServerConfig struct {
	Port           string
	AllowedOrigins []string // カンマ区切りで複数オリジンを受け入れる
	RateLimit      float64  // requests/sec
}

type DBConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type JWTConfig struct {
	Secret       []byte
	ExpiresHours int
}

// Load は環境変数を読み込んで Config を返す。
// 必須変数が未設定の場合はエラーを返す。
func Load() (*Config, error) {
	jwtSecret, err := requireEnv("JWT_SECRET")
	if err != nil {
		return nil, err
	}
	dbPassword, err := requireEnv("DB_PASSWORD")
	if err != nil {
		return nil, err
	}

	originsRaw := getEnv("ALLOWED_ORIGINS", "http://localhost:3000")
	origins := strings.Split(originsRaw, ",")
	for i, o := range origins {
		origins[i] = strings.TrimSpace(o)
	}

	return &Config{
		Server: ServerConfig{
			Port:           getEnv("SERVER_PORT", "8080"),
			AllowedOrigins: origins,
			RateLimit:      float64(getEnvInt("RATE_LIMIT", 20)),
		},
		DB: DBConfig{
			Host:            getEnv("DB_HOST", "127.0.0.1"),
			Port:            getEnv("DB_PORT", "15432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        dbPassword,
			Name:            getEnv("DB_NAME", "todo"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_MIN", 5)) * time.Minute,
		},
		JWT: JWTConfig{
			Secret:       []byte(jwtSecret),
			ExpiresHours: getEnvInt("JWT_EXPIRES_HOURS", 72),
		},
	}, nil
}

// DSN はデータベース接続文字列を構築して返す
func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

// ---- ヘルパー関数 ----

func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("必須環境変数 %q が設定されていません", key)
	}
	return v, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

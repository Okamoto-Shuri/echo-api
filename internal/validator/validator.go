// internal/validator/validator.go
// バリデーションエラーを統一された HTTPError に変換する
package validator

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

// CustomValidator は Echo の Validator インターフェースを実装する
type CustomValidator struct {
	v *validator.Validate
}

// New は CustomValidator を生成する
func New() *CustomValidator {
	return &CustomValidator{v: validator.New()}
}

// Validate はリクエストボディのバリデーションを実行する。
// エラーがあれば 422 Unprocessable Entity を返す。
func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.v.Struct(i); err != nil {
		errors := make(map[string]string)
		for _, e := range err.(validator.ValidationErrors) {
			errors[e.Field()] = buildMessage(e)
		}
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]interface{}{
			"message": "バリデーションエラーが発生しました",
			"errors":  errors,
		})
	}
	return nil
}

func buildMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "必須項目です"
	case "min":
		return "最小" + e.Param() + "文字以上で入力してください"
	case "max":
		return "最大" + e.Param() + "文字以内で入力してください"
	default:
		return "入力値が不正です（" + e.Tag() + "）"
	}
}

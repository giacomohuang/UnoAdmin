package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/unoadmin/admin-service/internal/domain"
)

// errorBody 是前后端约定的错误响应体。
type errorBody struct {
	Error string `json:"error"`
}

// writeJSON 写入 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return
	}
}

// writeError 将领域错误翻译成 HTTP 状态码与错误体。
func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, statusFor(err), errorBody{Error: err.Error()})
}

// statusFor 统一定义领域错误到 HTTP 状态码的映射。
func statusFor(err error) int {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrInvalid):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

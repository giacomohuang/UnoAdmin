package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/unoadmin/admin-service/internal/domain"
)

// errorBody 是前后端约定的错误响应体。
type errorBody struct {
	Error string `json:"error"`
}

// writeError 将领域错误翻译成 HTTP 状态码与错误体，供处理器使用。
func writeError(c *gin.Context, err error) {
	c.JSON(statusFor(err), errorBody{Error: err.Error()})
}

// abortWithError 写出错误响应并终止中间件链，供中间件使用。
func abortWithError(c *gin.Context, err error) {
	c.AbortWithStatusJSON(statusFor(err), errorBody{Error: err.Error()})
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

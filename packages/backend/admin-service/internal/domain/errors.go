package domain

import "errors"

// 服务层统一错误哨兵，由 HTTP/gRPC 适配器翻译成对应的状态码。
var (
	ErrNotFound     = errors.New("resource not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrInvalid      = errors.New("invalid argument")
	ErrConflict     = errors.New("conflict")
)

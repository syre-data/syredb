package handler

import "github.com/labstack/echo/v5"

func requestId(c *echo.Context) string {
	return requestId(c)
}

type AppError struct {
	RequestId string
	Code      AppErrorCode
	Message   string
	Payload   any
}

type AppErrorCode string

const (
	APP_ERR_EMAIL_NOT_SENT              AppErrorCode = "USER_EMAIL_NOT_SENT"
	APP_ERR_USER_NOT_AUTHENTICATED      AppErrorCode = "USER_NOT_AUTHENTICATED"
	APP_ERR_COULD_NOT_AUTHENTICATE_USER AppErrorCode = "COULD_NOT_AUTHENTICATE"
	APP_ERR_INVALID_PASSWORD            AppErrorCode = "INVALID_PASSWORD"
	APP_ERR_SESSION_NOT_DEACTIVATED     AppErrorCode = "SESSION_NOT_DEACTIVATED"
	APP_ERR_SESSION_TOKEN_NOT_CREATED   AppErrorCode = "SESSION_TOKEN_NOT_CREATED"
	APP_ERR_INVALID_SESSION_TOKEN       AppErrorCode = "INVALID_SESSION_TOKEN"
	APP_ERR_INVALID_UUID                AppErrorCode = "INVALID_UUID"
	APP_ERR_DB_QUERY                    AppErrorCode = "DB_QUERY"
	APP_ERR_RECORD_NOT_FOUND            AppErrorCode = "RECORD_NOT_FOUND"
	APP_ERR_ENTITY_NOT_FOUND            AppErrorCode = "ENTITY_NOT_FOUND"
	APP_ERR_INVALID_ITEM                AppErrorCode = "INVALID_ITEM"
	APP_ERR_INVALID_PAYLOAD             AppErrorCode = "INVALID_PAYLOAD"
	APP_ERR_FS_ERROR                    AppErrorCode = "FS_ERROR"
	APP_ERR_LOGICAL                     AppErrorCode = "ERR_LOGICAL"
	APP_ERR_OTHER                       AppErrorCode = "ERR_OTHER"
)

type DbErrorCode string

const (
	DB_ERR_DUPLICATE_RECORD DbErrorCode = "23505"
)

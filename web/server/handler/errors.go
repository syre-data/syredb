package handler

type AppError struct {
	Code    AppErrorCode
	Message string
	Payload any
}

type AppErrorCode string

const (
	AppErrCodeEmailNotSent           AppErrorCode = "USER_EMAIL_NOT_SENT"
	AppErrorCodeUserNotAuthenticated AppErrorCode = "USER_NOT_AUTHENTICATED"
	AppErrorCodeRecordNotFound       AppErrorCode = "RECORD_NOT_FOUND"
	AppErrorCodeInvalidPassword      AppErrorCode = "INVALID_PASSWORD"
)

type DBErrorCode string

const (
	DBErrCodeDuplicateRecord DBErrorCode = "23505"
)

package handler

type AppError struct {
	Code    AppErrorCode
	Message string
	Payload any
}

type AppErrorCode string

const (
	AppErrCodeUserWelcomeEmailNotSent AppErrorCode = "USER_WELCOME_EMAIL_NOT_SENT"
	AppErrorCodeUserNotAuthenticated  AppErrorCode = "USER_NOT_AUTHENTICATED"
	AppErrorCodeRecordNotFound        AppErrorCode = "RECORD_NOT_FOUND"
)

type DBErrorCode string

const (
	DBErrCodeDuplicateRecord DBErrorCode = "23505"
)

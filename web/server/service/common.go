package service

type InsufficientPermissionsError struct{}

func (e *InsufficientPermissionsError) Error() string {
	return "INSUFFICIENT_PERMISSIONS"
}

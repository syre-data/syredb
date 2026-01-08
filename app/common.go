package app

type RecordNotFoundError struct{}

func (e *RecordNotFoundError) Error() string {
	return "RECORD_NOT_FOUND"
}

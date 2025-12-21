package errors

import "net/http"

type HttpError struct {
	error
	Status int
}

func (e HttpError) Error() string {
	return http.StatusText(e.Status)
}

var ErrBadRequest = HttpError{
	Status: http.StatusBadRequest,
}
var ErrUnauthorized = HttpError{
	Status: http.StatusUnauthorized,
}
var ErrNotFound = HttpError{
	Status: http.StatusNotFound,
}
var ErrRequestedRangeNotSatisfiable = HttpError{
	Status: http.StatusRequestedRangeNotSatisfiable,
}
var ErrInternalServerError = HttpError{
	Status: http.StatusInternalServerError,
}

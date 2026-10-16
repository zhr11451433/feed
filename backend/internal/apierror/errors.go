package apierror

import (
	"errors"
	"net/http"

	"gorm.io/gorm"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrValidation   = errors.New("validation error")
)

func ClassifyHTTPStatus(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, gorm.ErrRecordNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

type Body struct {
	Error string `json:"error"`
}

func JSONBody(err error) Body {
	if err == nil {
		return Body{}
	}
	return Body{Error: err.Error()}
}

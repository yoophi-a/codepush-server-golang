package httperrors

import (
	"net/http"

	"github.com/yoophi/codepush-server-golang/internal/core/domain"
)

func Status(err error) int {
	switch err {
	case nil:
		return http.StatusOK
	case domain.ErrUnauthorized, domain.ErrExpired:
		return http.StatusUnauthorized
	case domain.ErrForbidden:
		return http.StatusForbidden
	case domain.ErrNotFound:
		return http.StatusNotFound
	case domain.ErrConflict:
		return http.StatusConflict
	case domain.ErrMalformedRequest:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

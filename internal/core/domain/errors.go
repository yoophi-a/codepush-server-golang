package domain

import "errors"

var (
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
	ErrMalformedRequest = errors.New("malformed request")
	ErrExpired          = errors.New("expired")
)

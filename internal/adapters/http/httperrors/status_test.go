package httperrors

import (
	"errors"
	"testing"

	"github.com/yoophi/codepush-server-golang/internal/core/domain"
)

func TestStatus(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{err: nil, want: 200},
		{err: domain.ErrUnauthorized, want: 401},
		{err: domain.ErrExpired, want: 401},
		{err: domain.ErrForbidden, want: 403},
		{err: domain.ErrNotFound, want: 404},
		{err: domain.ErrConflict, want: 409},
		{err: domain.ErrMalformedRequest, want: 400},
		{err: errors.New("boom"), want: 500},
	}
	for _, tc := range cases {
		if got := Status(tc.err); got != tc.want {
			t.Fatalf("Status(%v) = %d, want %d", tc.err, got, tc.want)
		}
	}
}

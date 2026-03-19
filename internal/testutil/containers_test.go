package testutil

import (
	"context"
	"strings"
	"testing"
)

func TestStartStack(t *testing.T) {
	prev := startStackFn
	defer func() { startStackFn = prev }()

	wantStack := &Stack{}
	wantEndpoints := Endpoints{
		DatabaseURL: "postgres://demo",
		RedisAddr:   "redis:6379",
		MinIOAddr:   "minio:9000",
	}
	startStackFn = func(context.Context) (*Stack, Endpoints, error) {
		return wantStack, wantEndpoints, nil
	}

	gotStack, gotEndpoints, err := StartStack(context.Background())
	if err != nil {
		t.Fatalf("StartStack() error = %v", err)
	}
	if gotStack != wantStack || gotEndpoints != wantEndpoints {
		t.Fatalf("unexpected StartStack() result stack=%#v endpoints=%#v", gotStack, gotEndpoints)
	}
}

func TestStartStackRecoversFromPanic(t *testing.T) {
	prev := startStackFn
	defer func() { startStackFn = prev }()

	startStackFn = func(context.Context) (*Stack, Endpoints, error) {
		panic("boom")
	}

	gotStack, gotEndpoints, err := StartStack(context.Background())
	if err == nil || !strings.Contains(err.Error(), "testcontainers unavailable") {
		t.Fatalf("expected recovered panic error, got %v", err)
	}
	if gotStack != nil || gotEndpoints != (Endpoints{}) {
		t.Fatalf("expected zero values after panic, got stack=%#v endpoints=%#v", gotStack, gotEndpoints)
	}
}

func TestStartStackReturnsUnderlyingError(t *testing.T) {
	prev := startStackFn
	defer func() { startStackFn = prev }()

	startStackFn = func(context.Context) (*Stack, Endpoints, error) {
		return nil, Endpoints{}, context.DeadlineExceeded
	}

	gotStack, gotEndpoints, err := StartStack(context.Background())
	if err == nil || err != context.DeadlineExceeded {
		t.Fatalf("expected underlying error, got %v", err)
	}
	if gotStack != nil || gotEndpoints != (Endpoints{}) {
		t.Fatalf("expected zero values on error, got stack=%#v endpoints=%#v", gotStack, gotEndpoints)
	}
}

func TestTerminateNilSafe(t *testing.T) {
	var stack Stack
	stack.Terminate(context.Background())
}

package core

import (
	"context"
	"errors"
	"testing"
)

func TestShutdownFunc_CanBeAssigned(t *testing.T) {
	// Verify ShutdownFunc can be assigned a function
	var fn ShutdownFunc = func(ctx context.Context) error {
		return nil
	}

	err := fn(context.Background())
	if err != nil {
		t.Errorf("ShutdownFunc returned unexpected error: %v", err)
	}
}

func TestShutdownFunc_PropagatesErrors(t *testing.T) {
	expectedErr := errors.New("shutdown error")
	var fn ShutdownFunc = func(ctx context.Context) error {
		return expectedErr
	}

	err := fn(context.Background())
	if err != expectedErr {
		t.Errorf("ShutdownFunc returned %v, want %v", err, expectedErr)
	}
}

func TestShutdownFunc_ReceivesContext(t *testing.T) {
	var receivedCtx context.Context
	var fn ShutdownFunc = func(ctx context.Context) error {
		receivedCtx = ctx
		return nil
	}

	ctx := context.WithValue(context.Background(), "key", "value")
	fn(ctx)

	if receivedCtx != ctx {
		t.Error("ShutdownFunc did not receive the expected context")
	}
}

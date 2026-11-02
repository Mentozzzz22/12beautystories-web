package database

import (
	"context"
	"errors"
	"testing"
	"time"
)

var errDown = errors.New("database is down")

func TestWaitForRetriesUntilPingSucceeds(t *testing.T) {
	calls := 0
	ping := func(context.Context) error {
		calls++
		if calls < 3 {
			return errDown
		}
		return nil
	}

	if err := waitFor(context.Background(), ping, time.Minute, time.Millisecond); err != nil {
		t.Fatalf("waitFor: %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestWaitForReturnsLastErrorAfterTimeout(t *testing.T) {
	ping := func(context.Context) error { return errDown }

	err := waitFor(context.Background(), ping, 20*time.Millisecond, time.Millisecond)
	if !errors.Is(err, errDown) {
		t.Errorf("err = %v, want %v", err, errDown)
	}
}

func TestWaitForStopsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	ping := func(context.Context) error {
		calls++
		cancel()
		return errDown
	}

	err := waitFor(ctx, ping, time.Minute, time.Minute)
	if !errors.Is(err, errDown) {
		t.Errorf("err = %v, want %v", err, errDown)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

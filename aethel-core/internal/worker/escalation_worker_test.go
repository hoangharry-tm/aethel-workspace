package worker

import (
	"context"
	"testing"
	"time"
)

func TestEscalationWorker_StopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	w := NewEscalationWorker(nil, time.Minute, nil)

	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
		// worker stopped as expected
	case <-time.After(100 * time.Millisecond):
		t.Error("escalation worker did not stop within 100ms after context cancel")
	}
}

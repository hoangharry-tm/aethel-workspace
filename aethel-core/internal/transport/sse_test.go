package transport

import (
	"testing"
	"time"
)

// Test 1: Subscribe + Publish delivers event to the subscribed channel.
func TestSubscribePublishDelivers(t *testing.T) {
	b := NewSSEBroker()
	ch, cleanup := b.Subscribe("user-1")
	defer cleanup()

	want := Event{Type: "notify", Data: "hello"}
	b.Publish("user-1", want)

	select {
	case got := <-ch:
		if got.Type != want.Type {
			t.Fatalf("got event type %q, want %q", got.Type, want.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for event")
	}
}

// Test 2: Multiple subscribers for the same userID all receive the published event.
func TestMultipleSubscribersReceiveEvent(t *testing.T) {
	b := NewSSEBroker()

	ch1, cleanup1 := b.Subscribe("user-2")
	defer cleanup1()
	ch2, cleanup2 := b.Subscribe("user-2")
	defer cleanup2()

	want := Event{Type: "ping", Data: nil}
	b.Publish("user-2", want)

	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case got := <-ch:
			if got.Type != want.Type {
				t.Fatalf("subscriber %d: got event type %q, want %q", i+1, got.Type, want.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("subscriber %d: timed out waiting for event", i+1)
		}
	}
}

// Test 3: Cleanup function removes the subscriber so no further events are received.
func TestCleanupRemovesSubscriber(t *testing.T) {
	b := NewSSEBroker()
	ch, cleanup := b.Subscribe("user-3")

	// Call cleanup immediately — subscriber should be removed.
	cleanup()

	// Publish should be a no-op (channel is closed, no subscribers).
	b.Publish("user-3", Event{Type: "after-cleanup", Data: nil})

	// The channel should be closed; reading from it must not block.
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected channel to be closed, but received a value")
		}
		// Channel closed as expected.
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out — closed channel read should not block")
	}

	// Verify the subscriber map no longer has an entry for this user.
	b.mu.RLock()
	subs := b.subscribers["user-3"]
	b.mu.RUnlock()
	if len(subs) != 0 {
		t.Fatalf("expected 0 subscribers after cleanup, got %d", len(subs))
	}
}

// Test 4: Publish to a userID with no subscribers is a no-op and does not panic.
func TestPublishToUnsubscribedUserNoPanic(t *testing.T) {
	b := NewSSEBroker()
	// Must not panic.
	b.Publish("nobody", Event{Type: "ghost", Data: nil})
}

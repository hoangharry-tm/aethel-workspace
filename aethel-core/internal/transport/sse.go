package transport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Event is a single SSE event sent to a connected client.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type subscriber struct {
	ch     chan Event
	userID string
}

// SSEBroker manages open SSE connections and broadcasts events to subscribers.
type SSEBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]*subscriber
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		subscribers: make(map[string][]*subscriber),
	}
}

// Publish sends an event to all connections for the given user ID.
func (b *SSEBroker) Publish(userID string, event Event) {
	b.mu.RLock()
	subs := b.subscribers[userID]
	b.mu.RUnlock()

	for _, sub := range subs {
		select {
		case sub.ch <- event:
		default:
			// Drop if the channel is full to avoid blocking the caller.
		}
	}
}

// Subscribe registers a new channel for userID.
// Returns the receive-only channel and a cleanup function that must be called
// (typically via defer) to remove the subscriber and close the channel.
func (b *SSEBroker) Subscribe(userID string) (<-chan Event, func()) {
	ch := make(chan Event, 64)
	sub := &subscriber{ch: ch, userID: userID}

	b.mu.Lock()
	b.subscribers[userID] = append(b.subscribers[userID], sub)
	b.mu.Unlock()

	cleanup := func() {
		b.mu.Lock()
		subs := b.subscribers[userID]
		for i, s := range subs {
			if s == sub {
				b.subscribers[userID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		if len(b.subscribers[userID]) == 0 {
			delete(b.subscribers, userID)
		}
		b.mu.Unlock()
		close(ch)
	}

	return ch, cleanup
}

// ServeHTTP handles an SSE connection for the authenticated user.
// The userID must be set on the request context before this handler runs.
func (b *SSEBroker) ServeHTTP(w http.ResponseWriter, r *http.Request, userID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sub := &subscriber{
		ch:     make(chan Event, 16),
		userID: userID,
	}

	b.mu.Lock()
	b.subscribers[userID] = append(b.subscribers[userID], sub)
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		subs := b.subscribers[userID]
		for i, s := range subs {
			if s == sub {
				b.subscribers[userID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		if len(b.subscribers[userID]) == 0 {
			delete(b.subscribers, userID)
		}
		b.mu.Unlock()
		close(sub.ch)
	}()

	// Send a ping to establish the connection.
	fmt.Fprintf(w, "event: ping\ndata: {}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-sub.ch:
			if !ok {
				return
			}
			payload, err := json.Marshal(event.Data)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, payload)
			flusher.Flush()
		}
	}
}

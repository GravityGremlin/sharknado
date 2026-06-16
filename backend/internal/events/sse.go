package events

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// EventBroker manages Server-Sent Events clients and broadcasting.
type EventBroker struct {
	mu      sync.RWMutex
	clients map[string]chan []byte
}

// NewEventBroker creates a new SSE event broker.
func NewEventBroker() *EventBroker {
	return &EventBroker{
		clients: make(map[string]chan []byte),
	}
}

// Subscribe registers a new client and returns a channel for events.
// The caller should listen on the channel and write events to the SSE response.
func (b *EventBroker) Subscribe() (string, <-chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := fmt.Sprintf("client-%d", len(b.clients)+1)
	ch := make(chan []byte, 64)
	b.clients[id] = ch
	return id, ch
}

// Unsubscribe removes a client by ID and closes its channel.
func (b *EventBroker) Unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.clients[id]; ok {
		close(ch)
		delete(b.clients, id)
	}
}

// Broadcast sends an event to all connected clients.
func (b *EventBroker) Broadcast(event string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("SSE marshal error: %v", err)
		return
	}

	msg := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, string(payload)))

	b.mu.RLock()
	defer b.mu.RUnlock()

	for id, ch := range b.clients {
		select {
		case ch <- msg:
		default:
			// Client too slow; drop message for this client
			log.Printf("SSE client %s too slow, dropping message", id)
		}
	}
}

// ServeHTTP handles SSE connections.
func (b *EventBroker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	id, ch := b.Subscribe()
	defer b.Unsubscribe(id)

	// Send initial connected event
	fmt.Fprintf(w, "event: connected\ndata: {\"client_id\": %q}\n\n", id)
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			_, _ = w.Write(msg)
			flusher.Flush()
		}
	}
}

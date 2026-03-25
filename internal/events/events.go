package events

import (
	"encoding/json"
	"log"
	"sync"
)

// Event types broadcast to SSE clients
const (
	EventNewArticles = "new_articles"
)

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

type Broadcaster struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		clients: make(map[chan []byte]struct{}),
	}
}

// Add registers a new SSE client channel
func (b *Broadcaster) Add() chan []byte {
	ch := make(chan []byte, 64)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Remove unregisters a client channel
func (b *Broadcaster) Remove(ch chan []byte) {
	b.mu.Lock()
	delete(b.clients, ch)
	close(ch)
	b.mu.Unlock()
}

// Broadcast sends an event to all connected clients
func (b *Broadcaster) Broadcast(eventType string, payload interface{}) {
	data, err := json.Marshal(Event{Type: eventType, Payload: payload})
	if err != nil {
		log.Printf("broadcast event encode error: %v", err)
		return
	}

	b.mu.RLock()
	for ch := range b.clients {
		select {
		case ch <- data:
		default:
			// client buffer full, skip
		}
	}
	b.mu.RUnlock()
}

// ClientCount returns the number of connected SSE clients
func (b *Broadcaster) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// Global broadcaster instance
var GlobalBroadcaster = NewBroadcaster()

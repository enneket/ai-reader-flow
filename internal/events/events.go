package events

import (
	"encoding/json"
	"log"
	"sync"
)

// Verify types implement the payloads used in SSE events

// Event types broadcast to SSE clients
const (
	EventNewArticles = "new_articles"

	// Refresh events
	EventRefreshStart    = "refresh:start"
	EventRefreshProgress = "refresh:progress"
	EventRefreshComplete = "refresh:complete"
	EventRefreshError    = "refresh:error"

	// Briefing events
	EventBriefingStart    = "briefing:start"
	EventBriefingProgress = "briefing:progress"
	EventBriefingComplete = "briefing:complete"
	EventBriefingError    = "briefing:error"
)

// Refresh progress payload
type RefreshProgress struct {
	Current   int    `json:"current"`
	Total     int    `json:"total"`
	FeedTitle string `json:"feedTitle"`
}

// Refresh complete payload
type RefreshComplete struct {
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

// Briefing progress payload
type BriefingProgress struct {
	Stage  string `json:"stage"`
	Detail string `json:"detail"`
}

// Briefing complete payload
type BriefingComplete struct {
	BriefingID int64 `json:"briefingId"`
}

// OperationState provides mutual exclusion between refresh and briefing operations
type OperationState struct {
	mutex   sync.Mutex
	current string // "idle" | "refreshing" | "generating"
}

// TryLock attempts to acquire the operation lock for the given operation name.
// Returns false if another operation is already in progress.
func (s *OperationState) TryLock(op string) bool {
	s.mutex.Lock()
	if s.current != "idle" {
		s.mutex.Unlock()
		return false
	}
	s.current = op
	s.mutex.Unlock()
	return true
}

// Unlock releases the current operation lock
func (s *OperationState) Unlock() {
	s.mutex.Lock()
	s.current = "idle"
	s.mutex.Unlock()
}

// Global operation state instance
var GlobalOperationState = &OperationState{}

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

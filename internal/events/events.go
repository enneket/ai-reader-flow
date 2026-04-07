package events

import (
	"encoding/json"
	"fmt"
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
	FeedId    int64  `json:"feedId"`
	NewCount  int    `json:"newCount"`
	Error     string `json:"error"`
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
	if s.current != "" && s.current != "idle" {
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

// Current returns the current operation name ("idle", "refreshing", "generating")
func (s *OperationState) Current() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.current
}

// Global operation state instance
var GlobalOperationState = &OperationState{}

// FeedRefreshResult holds per-feed refresh result
type FeedRefreshResult struct {
	FeedID   int64  `json:"feedId"`
	Title    string `json:"title"`
	Success  bool   `json:"success"`
	NewCount int    `json:"newCount"` // -1 表示失败
	Error    string `json:"error"`
}

// RefreshStatus holds the current refresh progress state
type RefreshStatus struct {
	Mutex      sync.Mutex
	InProgress bool
	Current    int
	Total      int
	FeedTitle  string
	Success    int
	Failed     int
	Error      string
	Results    map[int64]FeedRefreshResult // 新增
}

var GlobalRefreshStatus = &RefreshStatus{}

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// ProgressResponse is the JSON payload returned by GET /api/progress
type ProgressResponse struct {
	Operation string           `json:"operation"` // "idle" | "refreshing" | "generating"
	Refresh   *RefreshStatusDTO `json:"refresh,omitempty"`
}

// RefreshStatusDTO mirrors GlobalRefreshStatus for JSON serialization
type RefreshStatusDTO struct {
	InProgress bool   `json:"inProgress"`
	Current    int    `json:"current"`
	Total      int    `json:"total"`
	FeedTitle  string `json:"feedTitle"`
	Success    int    `json:"success"`
	Failed     int    `json:"failed"`
	Error      string `json:"error"`
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

// Broadcast sends an event to all connected clients in SSE format
func (b *Broadcaster) Broadcast(eventType string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("broadcast event encode error: %v", err)
		return
	}

	// SSE format: "event: <type>\r\ndata: <json>\r\n\r\n"
	message := fmt.Sprintf("event: %s\r\ndata: %s\r\n\r\n", eventType, data)

	b.mu.RLock()
	for ch := range b.clients {
		select {
		case ch <- []byte(message):
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

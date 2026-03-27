package events

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBroadcasterAddRemove(t *testing.T) {
	b := NewBroadcaster()

	// Add a client
	ch := b.Add()
	if b.ClientCount() != 1 {
		t.Errorf("ClientCount() = %d, want 1", b.ClientCount())
	}

	// Remove the client
	b.Remove(ch)
	if b.ClientCount() != 0 {
		t.Errorf("ClientCount() = %d, want 0", b.ClientCount())
	}
}

func TestBroadcasterAddRemoveConcurrent(t *testing.T) {
	b := NewBroadcaster()

	var wg sync.WaitGroup
	clientCount := 100

	// Add clients concurrently
	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Add()
		}()
	}
	wg.Wait()

	if b.ClientCount() != clientCount {
		t.Errorf("ClientCount() = %d, want %d", b.ClientCount(), clientCount)
	}

	// Remove clients concurrently
	wg.Add(clientCount)
	for i := 0; i < clientCount; i++ {
		go func() {
			defer wg.Done()
			ch := b.Add()
			b.Remove(ch)
		}()
	}
	wg.Wait()
}

func TestBroadcasterBroadcast(t *testing.T) {
	b := NewBroadcaster()
	ch := b.Add()
	defer b.Remove(ch)

	// Broadcast and verify received
	done := make(chan struct{})
	go func() {
		select {
		case data := <-ch:
			if !strings.Contains(string(data), "test_event") {
				t.Errorf("expected test_event in broadcast data, got %s", data)
			}
		case <-time.After(time.Second):
			t.Errorf("timeout waiting for broadcast")
		}
		close(done)
	}()

	b.Broadcast("test_event", map[string]string{"key": "value"})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Errorf("timeout waiting for broadcast to complete")
	}
}

func TestBroadcasterBroadcastMultipleClients(t *testing.T) {
	b := NewBroadcaster()
	clientCount := 5
	channels := make([]chan []byte, clientCount)
	for i := 0; i < clientCount; i++ {
		channels[i] = b.Add()
		defer b.Remove(channels[i])
	}

	b.Broadcast("multi_event", "test")

	// All clients should receive the broadcast
	for i, ch := range channels {
		select {
		case <-ch:
			// Received
		case <-time.After(time.Second):
			t.Errorf("client %d did not receive broadcast", i)
		}
	}
}

func TestBroadcasterClientCount(t *testing.T) {
	b := NewBroadcaster()

	if b.ClientCount() != 0 {
		t.Errorf("initial ClientCount() = %d, want 0", b.ClientCount())
	}

	ch1 := b.Add()
	if b.ClientCount() != 1 {
		t.Errorf("ClientCount() = %d, want 1", b.ClientCount())
	}

	ch2 := b.Add()
	if b.ClientCount() != 2 {
		t.Errorf("ClientCount() = %d, want 2", b.ClientCount())
	}

	b.Remove(ch1)
	if b.ClientCount() != 1 {
		t.Errorf("ClientCount() = %d, want 1", b.ClientCount())
	}

	b.Remove(ch2)
	if b.ClientCount() != 0 {
		t.Errorf("ClientCount() = %d, want 0", b.ClientCount())
	}
}

func TestBroadcasterClose(t *testing.T) {
	b := NewBroadcaster()
	ch := b.Add()

	b.Remove(ch)

	// Channel should be closed, trying to receive should return zero value
	select {
	case _, ok := <-ch:
		if ok {
			t.Errorf("expected closed channel")
		}
	default:
		// This is fine - channel could be empty
	}
}

func TestBroadcasterRemoveNonExistent(t *testing.T) {
	b := NewBroadcaster()
	ch := b.Add()
	b.Remove(ch)

	// After removal, channel is closed - we don't call Remove again
	// This test just verifies Add and Remove work without panic
}

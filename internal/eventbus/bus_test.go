package eventbus

import (
	"sync"
	"testing"
)

func TestPubSub(t *testing.T) {
	bus := New()
	var received []Event
	var mu sync.Mutex

	bus.Subscribe(TopicInboundMessage, func(e Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})

	bus.Publish(TopicInboundMessage, "hello")
	bus.Publish(TopicInboundMessage, "world")

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 {
		t.Fatalf("expected 2 events, got %d", len(received))
	}
	if received[0].Payload != "hello" {
		t.Fatalf("expected 'hello', got %v", received[0].Payload)
	}
	if received[1].Payload != "world" {
		t.Fatalf("expected 'world', got %v", received[1].Payload)
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := New()
	count := 0
	var mu sync.Mutex

	for i := 0; i < 3; i++ {
		bus.Subscribe(TopicError, func(e Event) {
			mu.Lock()
			count++
			mu.Unlock()
		})
	}

	bus.Publish(TopicError, "test")

	mu.Lock()
	defer mu.Unlock()
	if count != 3 {
		t.Fatalf("expected 3, got %d", count)
	}
}

func TestUnsubscribedTopic(t *testing.T) {
	bus := New()
	// Should not panic
	bus.Publish(TopicAgentThink, "no subscribers")
}

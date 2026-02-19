package eventbus

import (
	"sync"
	"time"
)

// Bus is a simple in-process pub/sub event bus.
type Bus struct {
	mu       sync.RWMutex
	handlers map[Topic][]Handler
}

// New creates a new event bus.
func New() *Bus {
	return &Bus{
		handlers: make(map[Topic][]Handler),
	}
}

// Subscribe registers a handler for a topic.
func (b *Bus) Subscribe(topic Topic, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[topic] = append(b.handlers[topic], handler)
}

// Publish sends an event to all subscribers of the topic.
// Handlers are called synchronously in the order they were registered.
func (b *Bus) Publish(topic Topic, payload any) {
	b.mu.RLock()
	handlers := make([]Handler, len(b.handlers[topic]))
	copy(handlers, b.handlers[topic])
	b.mu.RUnlock()

	event := Event{
		Topic:     topic,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	for _, h := range handlers {
		h(event)
	}
}

// PublishAsync sends an event to all subscribers asynchronously.
func (b *Bus) PublishAsync(topic Topic, payload any) {
	b.mu.RLock()
	handlers := make([]Handler, len(b.handlers[topic]))
	copy(handlers, b.handlers[topic])
	b.mu.RUnlock()

	event := Event{
		Topic:     topic,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	for _, h := range handlers {
		go h(event)
	}
}

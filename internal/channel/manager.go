package channel

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// Manager manages the lifecycle of all channels.
type Manager struct {
	mu       sync.RWMutex
	channels map[string]Channel
}

// NewManager creates a new channel manager.
func NewManager() *Manager {
	return &Manager{
		channels: make(map[string]Channel),
	}
}

// Register adds a channel to the manager.
func (m *Manager) Register(ch Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[ch.Name()] = ch
}

// StartAll starts all registered channels.
func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, ch := range m.channels {
		if err := ch.Start(ctx); err != nil {
			log.Printf("[channel] failed to start %s: %v", name, err)
			return fmt.Errorf("start %s: %w", name, err)
		}
		log.Printf("[channel] started %s", name)
	}
	return nil
}

// StopAll stops all running channels.
func (m *Manager) StopAll(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, ch := range m.channels {
		if ch.IsRunning() {
			if err := ch.Stop(ctx); err != nil {
				log.Printf("[channel] failed to stop %s: %v", name, err)
			} else {
				log.Printf("[channel] stopped %s", name)
			}
		}
	}
}

// Get returns a channel by name.
func (m *Manager) Get(name string) (Channel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ch, ok := m.channels[name]
	return ch, ok
}

// List returns all channel names and their running status.
func (m *Manager) List() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]bool, len(m.channels))
	for name, ch := range m.channels {
		result[name] = ch.IsRunning()
	}
	return result
}

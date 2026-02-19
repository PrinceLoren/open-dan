package channel

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// ConsoleChannel is a debug channel that reads from stdin and writes to stdout.
type ConsoleChannel struct {
	mu      sync.Mutex
	handler func(InboundMessage)
	running bool
	cancel  context.CancelFunc
}

func NewConsoleChannel() *ConsoleChannel {
	return &ConsoleChannel{}
}

func (c *ConsoleChannel) Name() string { return "console" }

func (c *ConsoleChannel) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.running = true

	go c.readLoop(ctx)
	return nil
}

func (c *ConsoleChannel) Stop(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}
	c.running = false
	return nil
}

func (c *ConsoleChannel) Send(_ context.Context, msg OutboundMessage) error {
	fmt.Printf("\n[OpenDan]: %s\n\n> ", msg.Text)
	return nil
}

func (c *ConsoleChannel) OnMessage(handler func(InboundMessage)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handler = handler
}

func (c *ConsoleChannel) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

func (c *ConsoleChannel) readLoop(ctx context.Context) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if scanner.Scan() {
				text := scanner.Text()
				if text == "" {
					fmt.Print("> ")
					continue
				}

				c.mu.Lock()
				handler := c.handler
				c.mu.Unlock()

				if handler != nil {
					handler(InboundMessage{
						ChannelName: "console",
						SenderID:    "local",
						SenderName:  "User",
						ChatID:      "console",
						Text:        text,
						Timestamp:   time.Now(),
					})
				}
			}
		}
	}
}

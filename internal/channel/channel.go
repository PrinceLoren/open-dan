package channel

import (
	"context"
	"time"
)

// InboundMessage is a message received from a channel.
type InboundMessage struct {
	ChannelName string
	SenderID    string
	SenderName  string
	ChatID      string
	Text        string
	Timestamp   time.Time
}

// OutboundMessage is a message to send through a channel.
type OutboundMessage struct {
	ChatID  string
	Text    string
	ReplyTo string // optional message ID to reply to
}

// Channel is the interface for messaging integrations.
type Channel interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Send(ctx context.Context, msg OutboundMessage) error
	OnMessage(handler func(InboundMessage))
	IsRunning() bool
}

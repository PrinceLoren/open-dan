package agent

import (
	"context"
	"log"
	"sync"

	"open-dan/internal/channel"
	"open-dan/internal/config"
	"open-dan/internal/eventbus"
	"open-dan/internal/llm"
	"open-dan/internal/memory"
	"open-dan/internal/tool"
)

// Agent is the core AI agent that processes messages through the think→act→observe loop.
type Agent struct {
	mu         sync.RWMutex
	cfg        config.AgentConfig
	provider   llm.Provider
	tools      *tool.Registry
	memory     memory.Memory
	bus        *eventbus.Bus
	chanMgr    *channel.Manager
	ctxManager *contextManager
}

// New creates a new Agent.
func New(
	cfg config.AgentConfig,
	provider llm.Provider,
	tools *tool.Registry,
	mem memory.Memory,
	bus *eventbus.Bus,
	chanMgr *channel.Manager,
) *Agent {
	return &Agent{
		cfg:        cfg,
		provider:   provider,
		tools:      tools,
		memory:     mem,
		bus:        bus,
		chanMgr:    chanMgr,
		ctxManager: newContextManager(provider, cfg.ContextWindow, cfg.SummarizeAt),
	}
}

// Start begins listening for inbound messages from all channels.
func (a *Agent) Start(ctx context.Context) {
	// Wire up all channels to route messages to the agent
	for name, running := range a.chanMgr.List() {
		if !running {
			continue
		}
		ch, ok := a.chanMgr.Get(name)
		if !ok {
			continue
		}
		ch.OnMessage(func(msg channel.InboundMessage) {
			a.bus.Publish("inbound_message", msg)
			a.handleMessage(ctx, msg)
		})
	}

	log.Println("[agent] started and listening for messages")
}

// handleMessage processes an inbound message and sends the response back.
func (a *Agent) handleMessage(ctx context.Context, msg channel.InboundMessage) {
	log.Printf("[agent] processing message from %s (%s): %s", msg.SenderName, msg.ChannelName, truncate(msg.Text, 100))

	response, err := a.processMessage(ctx, msg.ChatID, msg.Text)
	if err != nil {
		log.Printf("[agent] error processing message: %v", err)
		response = "Sorry, I encountered an error processing your message. Please try again."
		a.bus.Publish("error", err)
	}

	// Send response back through the channel
	ch, ok := a.chanMgr.Get(msg.ChannelName)
	if !ok {
		log.Printf("[agent] channel %s not found", msg.ChannelName)
		return
	}

	outMsg := channel.OutboundMessage{
		ChatID: msg.ChatID,
		Text:   response,
	}
	a.bus.Publish("outbound_message", outMsg)

	if err := ch.Send(ctx, outMsg); err != nil {
		log.Printf("[agent] error sending response: %v", err)
	}
}

// HandleDirectMessage processes a message from the GUI directly.
func (a *Agent) HandleDirectMessage(ctx context.Context, chatID, text string) (string, error) {
	return a.processMessage(ctx, chatID, text)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

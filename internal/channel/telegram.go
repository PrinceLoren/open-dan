package channel

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	tele "gopkg.in/telebot.v3"
)

// TelegramChannel integrates with the Telegram Bot API.
type TelegramChannel struct {
	mu         sync.Mutex
	token      string
	allowedIDs map[int64]bool
	bot        *tele.Bot
	handler    func(InboundMessage)
	running    bool
}

// TelegramConfig holds Telegram-specific configuration.
type TelegramConfig struct {
	Token      string
	AllowedIDs []int64
}

// NewTelegramChannel creates a new Telegram channel.
func NewTelegramChannel(cfg TelegramConfig) *TelegramChannel {
	allowed := make(map[int64]bool, len(cfg.AllowedIDs))
	for _, id := range cfg.AllowedIDs {
		allowed[id] = true
	}
	return &TelegramChannel{
		token:      cfg.Token,
		allowedIDs: allowed,
	}
}

func (t *TelegramChannel) Name() string { return "telegram" }

func (t *TelegramChannel) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return nil
	}

	pref := tele.Settings{
		Token:  t.token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		return fmt.Errorf("telegram bot init: %w", err)
	}

	bot.Handle(tele.OnText, func(c tele.Context) error {
		sender := c.Sender()

		// Authorization check
		if len(t.allowedIDs) > 0 && !t.allowedIDs[sender.ID] {
			log.Printf("[telegram] unauthorized user: %d (%s)", sender.ID, sender.Username)
			return nil // silently ignore
		}

		t.mu.Lock()
		handler := t.handler
		t.mu.Unlock()

		if handler != nil {
			handler(InboundMessage{
				ChannelName: "telegram",
				SenderID:    strconv.FormatInt(sender.ID, 10),
				SenderName:  sender.FirstName + " " + sender.LastName,
				ChatID:      strconv.FormatInt(c.Chat().ID, 10),
				Text:        c.Text(),
				Timestamp:   time.Now(),
			})
		}
		return nil
	})

	t.bot = bot
	t.running = true

	go func() {
		bot.Start()
	}()

	// Stop bot when context is cancelled
	go func() {
		<-ctx.Done()
		bot.Stop()
	}()

	return nil
}

func (t *TelegramChannel) Stop(_ context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.bot != nil {
		t.bot.Stop()
	}
	t.running = false
	return nil
}

func (t *TelegramChannel) Send(_ context.Context, msg OutboundMessage) error {
	t.mu.Lock()
	bot := t.bot
	t.mu.Unlock()

	if bot == nil {
		return fmt.Errorf("telegram bot not started")
	}

	chatID, err := strconv.ParseInt(msg.ChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	recipient := &tele.Chat{ID: chatID}

	// Split long messages (Telegram limit is 4096)
	text := msg.Text
	for len(text) > 0 {
		chunk := text
		if len(chunk) > 4000 {
			chunk = text[:4000]
			text = text[4000:]
		} else {
			text = ""
		}
		if _, err := bot.Send(recipient, chunk); err != nil {
			return fmt.Errorf("telegram send: %w", err)
		}
	}

	return nil
}

func (t *TelegramChannel) OnMessage(handler func(InboundMessage)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handler = handler
}

func (t *TelegramChannel) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

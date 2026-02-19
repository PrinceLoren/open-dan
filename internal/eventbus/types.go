package eventbus

import "time"

// Topic represents an event topic.
type Topic string

const (
	TopicInboundMessage  Topic = "inbound_message"
	TopicOutboundMessage Topic = "outbound_message"
	TopicAgentThink      Topic = "agent_think"
	TopicAgentAct        Topic = "agent_act"
	TopicAgentObserve    Topic = "agent_observe"
	TopicToolCall        Topic = "tool_call"
	TopicToolResult      Topic = "tool_result"
	TopicLLMRequest      Topic = "llm_request"
	TopicLLMResponse     Topic = "llm_response"
	TopicError           Topic = "error"
	TopicStatusChange    Topic = "status_change"
)

// Event is a message passed through the event bus.
type Event struct {
	Topic     Topic
	Payload   any
	Timestamp time.Time
}

// Handler processes an event.
type Handler func(Event)

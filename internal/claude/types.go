// Package claude is an HTTP client for the Anthropic Messages API.
// It intentionally does not use any Anthropic SDK — direct HTTP helps
// with understanding the agentic loop mechanics.
package claude

import "encoding/json"

// Role is the author of a message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// StopReason indicates why the model stopped generating.
// This value drives the agentic loop: "tool_use" means we must
// execute tools and continue, "end_turn" means the turn is complete.
type StopReason string

const (
	StopReasonEndTurn   StopReason = "end_turn"
	StopReasonToolUse   StopReason = "tool_use"
	StopReasonMaxTokens StopReason = "max_tokens"
	StopReasonStopSeq   StopReason = "stop_sequence"
)

// ContentBlockType identifies the variant of a content block.
type ContentBlockType string

const (
	ContentTypeText       ContentBlockType = "text"
	ContentTypeToolUse    ContentBlockType = "tool_use"
	ContentTypeToolResult ContentBlockType = "tool_result"
	ContentTypeDocument   ContentBlockType = "document"
)

// ContentBlock represents a single element in a message's content array.
// Claude API content is polymorphic: a message may mix text, tool calls,
// tool results, and documents. Unused fields are omitted via omitempty.
type ContentBlock struct {
	Type ContentBlockType `json:"type"`

	// Text block fields.
	Text string `json:"text,omitempty"`

	// Tool use block fields (assistant requesting a tool call).
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// Tool result block fields (user providing tool execution result).
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`

	// Document block fields.
	Source *DocumentSource `json:"source,omitempty"`
}

// DocumentSource describes an inline document attached to a user message.
type DocumentSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "application/pdf"
	Data      string `json:"data"`       // base64-encoded bytes
}

// Message is a single turn in the conversation.
type Message struct {
	Role    Role           `json:"role"`
	Content []ContentBlock `json:"content"`
}

// Tool is a tool definition sent to the API.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// MessagesRequest is the payload for POST /v1/messages.
type MessagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
	Tools     []Tool    `json:"tools,omitempty"`
}

// MessagesResponse is the response payload from POST /v1/messages.
type MessagesResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         Role           `json:"role"`
	Model        string         `json:"model"`
	Content      []ContentBlock `json:"content"`
	StopReason   StopReason     `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence,omitempty"`
	Usage        Usage          `json:"usage"`
}

// Usage reports token consumption for a request.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// APIError is the error shape returned by Anthropic on 4xx/5xx.
type APIError struct {
	StatusCode int    `json:"-"`
	Type       string `json:"type"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return "anthropic api: " + e.Type + ": " + e.Message
}

package claude

import (
	"context"
	"encoding/json"
	"fmt"
)

// ToolHandler executes a single tool invocation. The input is the raw JSON
// from the model's tool_use block. The returned string is sent back as
// tool_result content. If err is non-nil, the result is marked isError=true
// so the model can reason about the failure.
type ToolHandler func(ctx context.Context, input json.RawMessage) (string, error)

// ToolRegistry holds tool definitions and their executors.
// This separation lets us change tool behavior without touching API schemas.
type ToolRegistry struct {
	defs     map[string]Tool
	handlers map[string]ToolHandler
}

// NewToolRegistry creates an empty registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		defs:     make(map[string]Tool),
		handlers: make(map[string]ToolHandler),
	}
}

// Register adds a tool with its handler. Name must match Tool.Name.
func (r *ToolRegistry) Register(tool Tool, handler ToolHandler) error {
	if tool.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if _, exists := r.defs[tool.Name]; exists {
		return fmt.Errorf("tool %q already registered", tool.Name)
	}
	r.defs[tool.Name] = tool
	r.handlers[tool.Name] = handler
	return nil
}

// Definitions returns the tool definitions for inclusion in a request.
func (r *ToolRegistry) Definitions() []Tool {
	out := make([]Tool, 0, len(r.defs))
	for _, t := range r.defs {
		out = append(out, t)
	}
	return out
}

// Execute runs the handler for a tool call. Returns the result content and
// whether the tool itself reported an error (for the isError flag).
func (r *ToolRegistry) Execute(ctx context.Context, name string, input json.RawMessage) (string, bool) {
	handler, ok := r.handlers[name]
	if !ok {
		return fmt.Sprintf("unknown tool: %s", name), true
	}
	result, err := handler(ctx, input)
	if err != nil {
		return err.Error(), true
	}
	return result, false
}

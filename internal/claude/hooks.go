package claude

import (
	"context"
	"encoding/json"
)

// PostToolUseHook runs after a tool handler completes, before its result
// is added to conversation history. Hooks can inspect inputs/outputs and
// perform deterministic transformations on application state.
//
// This corresponds to Task Statement 1.5 — using hooks for guarantees that
// prompt-based instructions cannot provide deterministically.
type PostToolUseHook func(ctx context.Context, call ToolCall) error

// ToolCall describes a single tool invocation, passed to hooks.
type ToolCall struct {
	Name    string          // The tool that was called
	Input   json.RawMessage // The model's input to the tool
	Output  string          // The tool's return string (sent back to model)
	IsError bool            // Whether the handler reported an error
}

// HookRegistry holds PostToolUse hooks indexed by tool name.
// A wildcard tool name "*" registers hooks that run for every tool call.
type HookRegistry struct {
	postToolUse map[string][]PostToolUseHook
}

// NewHookRegistry constructs an empty hook registry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{
		postToolUse: make(map[string][]PostToolUseHook),
	}
}

// AddPostToolUse registers a hook to run after the named tool completes.
// Pass toolName="*" to run for every tool.
func (r *HookRegistry) AddPostToolUse(toolName string, hook PostToolUseHook) {
	r.postToolUse[toolName] = append(r.postToolUse[toolName], hook)
}

// runPostToolUse executes all registered hooks for a tool call.
// Hooks for the specific tool name run first, then wildcard hooks.
// If any hook returns an error, the loop stops and the error propagates.
func (r *HookRegistry) runPostToolUse(ctx context.Context, call ToolCall) error {
	for _, hook := range r.postToolUse[call.Name] {
		if err := hook(ctx, call); err != nil {
			return err
		}
	}
	for _, hook := range r.postToolUse["*"] {
		if err := hook(ctx, call); err != nil {
			return err
		}
	}
	return nil
}

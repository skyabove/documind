package claude

import (
	"context"
	"fmt"
	"log/slog"
)

// RunAgentOptions configures a single agentic loop execution.
type RunAgentOptions struct {
	// System prompt prepended to the conversation.
	System string

	// Initial user message content (text, documents, etc).
	InitialContent []ContentBlock

	// Tool registry available to the agent.
	Tools *ToolRegistry

	// MaxIterations is a safety cap, NOT the primary termination mechanism.
	// Termination is driven by stop_reason == "end_turn".
	// If the loop hits this cap, something is wrong (infinite tool calls).
	MaxIterations int

	// MaxTokens per API call.
	MaxTokens int
}

// AgentResult is the final state after the agentic loop completes.
type AgentResult struct {
	// FinalMessages is the full conversation history, useful for debugging.
	FinalMessages []Message

	// FinalText is the concatenated text content from the last assistant message.
	FinalText string

	// Iterations is how many API calls were made.
	Iterations int

	// StopReason from the final response.
	StopReason StopReason

	// TotalUsage aggregates token usage across all iterations.
	TotalUsage Usage
}

// RunAgent executes the agentic loop:
//  1. Send messages to Claude
//  2. Inspect stop_reason
//  3. If "end_turn" → return result
//  4. If "tool_use" → execute all requested tools, append results, loop
//
// This is the exam-critical pattern from Task Statement 1.1.
func (c *Client) RunAgent(ctx context.Context, opts RunAgentOptions) (*AgentResult, error) {
	if opts.MaxIterations == 0 {
		opts.MaxIterations = 10
	}
	if opts.MaxTokens == 0 {
		opts.MaxTokens = 4096
	}

	messages := []Message{
		{Role: RoleUser, Content: opts.InitialContent},
	}

	result := &AgentResult{}

	for iter := 0; iter < opts.MaxIterations; iter++ {
		result.Iterations = iter + 1

		req := MessagesRequest{
			Model:     c.model,
			MaxTokens: opts.MaxTokens,
			System:    opts.System,
			Messages:  messages,
		}
		if opts.Tools != nil {
			req.Tools = opts.Tools.Definitions()
		}

		slog.InfoContext(ctx, "agent iteration",
			"iter", iter+1,
			"messages", len(messages),
		)

		resp, err := c.CreateMessage(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("iteration %d: %w", iter+1, err)
		}

		result.TotalUsage.InputTokens += resp.Usage.InputTokens
		result.TotalUsage.OutputTokens += resp.Usage.OutputTokens
		result.StopReason = resp.StopReason

		// Append the assistant's response to history.
		messages = append(messages, Message{
			Role:    RoleAssistant,
			Content: resp.Content,
		})

		// Termination: exam-critical branch.
		if resp.StopReason == StopReasonEndTurn {
			result.FinalMessages = messages
			result.FinalText = extractText(resp.Content)
			slog.InfoContext(ctx, "agent finished",
				"iterations", result.Iterations,
				"input_tokens", result.TotalUsage.InputTokens,
				"output_tokens", result.TotalUsage.OutputTokens,
			)
			return result, nil
		}

		// Only other case we handle: the model wants to call tools.
		if resp.StopReason != StopReasonToolUse {
			return nil, fmt.Errorf("unexpected stop_reason: %s", resp.StopReason)
		}

		if opts.Tools == nil {
			return nil, fmt.Errorf("model requested tool use but no tools were registered")
		}

		// Execute all tool calls in the response, collect tool_result blocks.
		toolResults := make([]ContentBlock, 0)
		for _, block := range resp.Content {
			if block.Type != ContentTypeToolUse {
				continue
			}
			slog.InfoContext(ctx, "tool call",
				"name", block.Name,
				"id", block.ID,
			)
			output, isErr := opts.Tools.Execute(ctx, block.Name, block.Input)
			toolResults = append(toolResults, ContentBlock{
				Type:      ContentTypeToolResult,
				ToolUseID: block.ID,
				Content:   output,
				IsError:   isErr,
			})
		}

		// Append tool results as a new user message for the next iteration.
		messages = append(messages, Message{
			Role:    RoleUser,
			Content: toolResults,
		})
	}

	return nil, fmt.Errorf("agent hit max iterations (%d) without reaching end_turn — possible infinite loop", opts.MaxIterations)
}

// extractText concatenates all text blocks from a content array.
func extractText(content []ContentBlock) string {
	var out string
	for _, b := range content {
		if b.Type == ContentTypeText {
			out += b.Text
		}
	}
	return out
}

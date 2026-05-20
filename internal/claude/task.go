package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// TaskInput is the input schema for the Task tool.
// The model emits this structure when it wants to delegate to a subagent.
type TaskInput struct {
	Agent  string `json:"agent"`
	Prompt string `json:"prompt"`
}

// taskInputSchemaTemplate is filled with the list of available agent names
// at registration time. The "agent" field uses an enum derived from
// AgentRegistry contents — the model cannot request a subagent that
// doesn't exist (enforced at API level, not by prompt).
const taskInputSchemaTemplate = `{
  "type": "object",
  "properties": {
    "agent": {
      "type": "string",
      "enum": %s,
      "description": "Name of the subagent to invoke. Each subagent is specialized for a particular task."
    },
    "prompt": {
      "type": "string",
      "description": "Complete instruction for the subagent. The subagent has NO access to the current conversation, so include all necessary context: what to investigate, what data to use, what output format you expect. Be specific about success criteria."
    }
  },
  "required": ["agent", "prompt"]
}`

// taskToolDescriptionTemplate is filled with a bullet list of available
// agents (name + their AgentDefinition.Description) at registration time.
const taskToolDescriptionTemplate = `Delegate a focused subtask to a specialized subagent. ` +
	`Use this when a task is best handled by an agent with specific tools or expertise. ` +
	`The subagent runs in an isolated context — it does NOT see this conversation, ` +
	`so the prompt must contain all needed information.

Available agents:
%s
When to use Task:
- The task requires tools/expertise outside this agent's role
- The task would produce verbose output that should be summarized before returning
- Multiple independent investigations can run in parallel by emitting several Task calls in one response

When NOT to use Task:
- Simple operations you can do directly with your own tools
- Tasks where you need ongoing back-and-forth — Task is a single round-trip`

// RegisterTaskTool installs the Task tool into the given ToolRegistry,
// wiring its handler to spawn subagents from the AgentRegistry.
//
// Parameters:
//   - client: used to run the nested RunAgent for spawned subagents
//   - toolReg: where Task tool gets registered (typically the coordinator's tools)
//   - agentReg: which subagents are available to delegate to
//   - sharedTools: pool of tools that subagents may use, filtered by AllowedTools
//
// The Task tool's input schema enum and description are built dynamically
// from agentReg contents — only registered agents are exposed to the model.
//
// Returns an error if no agents are registered (Task tool would be useless).
func RegisterTaskTool(client *Client, toolReg *ToolRegistry, agentReg *AgentRegistry, sharedTools *ToolRegistry) error {
	agentNames := agentReg.Names()
	if len(agentNames) == 0 {
		return fmt.Errorf("cannot register Task tool: no agents registered")
	}

	enumJSON, err := json.Marshal(agentNames)
	if err != nil {
		return fmt.Errorf("marshal agent names: %w", err)
	}

	var agentList string
	for _, name := range agentNames {
		def, _ := agentReg.Get(name)
		agentList += fmt.Sprintf("- %q: %s\n", name, def.Description)
	}

	tool := Tool{
		Name:        "Task",
		Description: fmt.Sprintf(taskToolDescriptionTemplate, agentList),
		InputSchema: json.RawMessage(fmt.Sprintf(taskInputSchemaTemplate, string(enumJSON))),
	}

	handler := func(ctx context.Context, input json.RawMessage) (string, error) {
		// Step 1: parse model's input.
		var ti TaskInput
		if err := json.Unmarshal(input, &ti); err != nil {
			return "", fmt.Errorf("invalid Task input: %w", err)
		}

		// Step 2: look up the agent definition.
		def, ok := agentReg.Get(ti.Agent)
		if !ok {
			return "", fmt.Errorf("unknown agent: %s", ti.Agent)
		}

		slog.InfoContext(ctx, "spawning subagent",
			"agent", ti.Agent,
			"prompt_chars", len(ti.Prompt),
		)

		// Step 3: build a tool registry containing ONLY this subagent's
		// allowed tools, drawn from the shared pool. Tools not in the
		// shared pool are silently skipped — this allows AgentDefinitions
		// to declare tool needs without forcing all tools to exist in
		// every context.
		subTools := NewToolRegistry()
		for _, toolName := range def.AllowedTools {
			if t, h, ok := sharedTools.lookup(toolName); ok {
				if err := subTools.Register(t, h); err != nil {
					return "", fmt.Errorf("register tool %s for subagent: %w", toolName, err)
				}
			}
		}

		// Step 4: run the subagent's own loop. Note: NO context inheritance.
		// The prompt is the entirety of what the subagent sees, besides its
		// own system prompt.
		result, err := client.RunAgent(ctx, RunAgentOptions{
			System: def.System,
			InitialContent: []ContentBlock{
				{Type: ContentTypeText, Text: ti.Prompt},
			},
			Tools:             subTools,
			InitialToolChoice: def.InitialToolChoice,
			MaxIterations:     def.MaxIterations,
		})
		if err != nil {
			return "", fmt.Errorf("subagent %s failed: %w", ti.Agent, err)
		}

		slog.InfoContext(ctx, "subagent finished",
			"agent", ti.Agent,
			"iterations", result.Iterations,
			"input_tokens", result.TotalUsage.InputTokens,
			"output_tokens", result.TotalUsage.OutputTokens,
		)

		// Step 5: return the subagent's final text as this tool's output.
		// It will become the content of a tool_result block in the parent
		// agent's conversation history.
		return result.FinalText, nil
	}

	return toolReg.Register(tool, handler)
}

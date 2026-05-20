package claude

import "errors"

// AgentDefinition describes a subagent that the Task tool can spawn.
//
// Each subagent has its own system prompt and a restricted tool set.
// Subagents do NOT inherit the parent agent's conversation history —
// the parent must pass relevant context explicitly via the Task prompt.
//
// This corresponds to Task Statement 1.3:
//   - "subagent context must be explicitly provided in the prompt"
//   - "AgentDefinition configuration including descriptions, system prompts,
//     and tool restrictions for each subagent type"
type AgentDefinition struct {
	// Name is the unique identifier the parent uses to invoke this subagent.
	Name string

	// Description tells the parent what this subagent does and when to use it.
	// This string is read by the LLM when deciding whether to delegate.
	Description string

	// System is the system prompt for the subagent's own conversation.
	System string

	// AllowedTools lists tool names available within the subagent's loop.
	// Empty/nil means no tools — subagent operates in pure-LLM mode.
	// Subagent CANNOT access tools not in this whitelist.
	AllowedTools []string

	// InitialToolChoice can force or require tool use on the subagent's
	// first turn only. This is useful for structured-output subagents:
	// force the recording tool once, then allow the second turn to end.
	InitialToolChoice *ToolChoice

	// MaxIterations caps the subagent's internal loop. Same semantics as
	// RunAgentOptions: a safety net against infinite tool-call loops,
	// not the primary termination mechanism.
	MaxIterations int
}

// AgentRegistry holds AgentDefinition entries indexed by name.
// The Task tool consults this registry when the model invokes a subagent.
type AgentRegistry struct {
	defs map[string]AgentDefinition
}

// NewAgentRegistry constructs an empty registry.
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{defs: make(map[string]AgentDefinition)}
}

// Register adds an agent definition. Names must be unique.
func (r *AgentRegistry) Register(def AgentDefinition) error {
	if def.Name == "" {
		return errEmptyAgentName
	}
	if _, exists := r.defs[def.Name]; exists {
		return errDuplicateAgent(def.Name)
	}
	r.defs[def.Name] = def
	return nil
}

// Get retrieves a definition by name. The bool indicates presence.
func (r *AgentRegistry) Get(name string) (AgentDefinition, bool) {
	def, ok := r.defs[name]
	return def, ok
}

// Names returns the registered agent names. Used by RegisterTaskTool to
// build the Task tool's input schema enum and description list.
func (r *AgentRegistry) Names() []string {
	out := make([]string, 0, len(r.defs))
	for name := range r.defs {
		out = append(out, name)
	}
	return out
}

var errEmptyAgentName = errors.New("agent name is required")

func errDuplicateAgent(name string) error {
	return errors.New("agent already registered: " + name)
}

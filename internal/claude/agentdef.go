package claude

import "errors"

// AgentDefinition describes a subagent that the Task tool can spawn.
//
// Each subagent has its own system prompt and a restricted tool set.
// Subagents do NOT inherit the parent agent's conversation history —
// the parent must pass relevant context explicitly via the prompt.
//
// This corresponds to Task Statement 1.3 in the certification:
//   - "subagent context must be explicitly provided in the prompt"
//   - "AgentDefinition configuration including descriptions, system prompts,
//     and tool restrictions for each subagent type"
type AgentDefinition struct {
	// Name is the unique identifier the parent agent uses to invoke this subagent.
	Name string

	// Description tells the parent what this subagent is for and when to use it.
	// This is what the LLM reads when deciding whether to delegate.
	Description string

	// System is the system prompt for the subagent's own conversation.
	System string

	// AllowedTools lists tool names available within the subagent's loop.
	// Empty means no tools — subagent operates in pure-LLM mode.
	// Subagent CANNOT use tools its parent has unless explicitly listed here.
	AllowedTools []string

	// MaxIterations caps the subagent's internal loop. Same semantics as in
	// RunAgentOptions: safety net, not primary termination.
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

// Names returns the registered agent names, useful for embedding in
// the Task tool's description so the LLM knows what's available.
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

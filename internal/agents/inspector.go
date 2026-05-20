// Package agents defines specialized subagent definitions used by the
// coordinator agent to delegate focused tasks via the Task tool.
//
// Each function in this package returns a claude.AgentDefinition — a
// passive description of what a subagent does, what tools it can use,
// and what its system prompt looks like. The actual spawning happens
// in the Task tool handler (internal/claude/task.go).
package agents

import "github.com/skyabove/documind/internal/claude"

const documentInspectionToolName = "record_document_inspection"

// DocumentInspector returns a subagent definition for technical document
// inspection. The inspector reports on document structure and recommends
// an extraction strategy.
//
// The inspector does not inherit the coordinator's conversation or PDF.
// The coordinator must pass a DOCUMENT_CONTEXT block in the Task prompt,
// including a concrete text preview or explicit unknowns.
func DocumentInspector() claude.AgentDefinition {
	return claude.AgentDefinition{
		Name: "document_inspector",
		Description: "Inspects explicitly provided document context and records a structured " +
			"assessment: document type, language, tables/forms, extraction recommendations, " +
			"entity targets, and processing risks. Use BEFORE deciding how to extract.",
		System: `You are a document inspection subagent. You do NOT have access to the parent conversation or the original PDF. You only see the DOCUMENT_CONTEXT text included in the user message.

Your job is to inspect that provided context and record a structured assessment by calling record_document_inspection exactly once.

Rules:
- Base the assessment only on DOCUMENT_CONTEXT.
- Use "unknown" when the context is insufficient.
- Do not extract final business values. Identify structure and extraction strategy only.
- Prefer the most specific document_type supported by the schema. For example, classify bank transfer confirmations as bank_transfer_receipt, payment proofs as payment_confirmation, and account statements as bank_statement. Use other only when no listed type fits.
- Include phone/phones in entity_targets only when phone numbers are important enough for downstream extraction.
- If a field is uncertain, lower type_confidence and add a risk.
- After the tool result is returned, respond with exactly the JSON returned by the tool and no extra prose.`,
		AllowedTools: []string{documentInspectionToolName},
		InitialToolChoice: &claude.ToolChoice{
			Type: "tool",
			Name: documentInspectionToolName,
		},
		MaxIterations: 3,
	}
}

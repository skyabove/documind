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
		System: `You are a document inspection subagent.

Your job is NOT to extract business facts.
Your job is to diagnose the document's type, structure, text quality, locale conventions, and processing risks.

You do not have access to the parent conversation or the original PDF.
You only see the DOCUMENT_CONTEXT text included in the user message.

Required behavior:
1. Read the provided DOCUMENT_CONTEXT.
2. Call record_document_inspection exactly once.
3. Fill the tool input using only structural and diagnostic observations.
4. Do not extract specific names, amounts, dates, account numbers, reference numbers, or phone numbers as final data.
5. After the tool result is returned, respond with exactly the JSON returned by the tool and no extra prose.

Field guidance:
- document_type: classify the document category, not its extracted contents.
- structure: use "form" for labeled key-value layouts; "table" for row/column data; "mixed" when both are important.
- layout_complexity: low for simple one-page receipts/forms; high for dense, multi-section, multi-column, or bundled documents.
- text_quality: good when fields are readable and coherent; partial when some fields are missing or uncertain; poor for OCR noise or broken ordering.
- approx_pages: estimate from context; use 1 if the context clearly represents a single-page document.
- contains_* fields: mark whether such information appears to exist, not what the specific values are.
- contains_masked_sensitive_data: true when account numbers, identifiers, or personal data are partially hidden.
- locale_format: infer only from visible formatting. Use es_ES for decimal comma and DD/MM/YYYY patterns.
- risks: return an empty array when no processing risks are observed. Do not use "none".

Do not duplicate the summary or entity extraction tools.
Do not recommend which extraction tools to call.
Do not include a notes field.`,
		AllowedTools: []string{documentInspectionToolName},
		InitialToolChoice: &claude.ToolChoice{
			Type: "tool",
			Name: documentInspectionToolName,
		},
		MaxIterations: 3,
	}
}

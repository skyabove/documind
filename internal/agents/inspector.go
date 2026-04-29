// Package agents defines specialized subagent definitions used by the
// coordinator agent to delegate focused tasks.
package agents

import "github.com/skyabove/documind/internal/claude"

// DocumentInspector returns a subagent definition for technical document
// inspection. The inspector reports on document structure (length, language,
// presence of tables/forms, document type confidence) without performing
// extraction itself.
//
// Use case: the coordinator can call this subagent to decide HOW to extract
// a document — e.g., a multi-page invoice with tables may need a different
// strategy than a one-page receipt.
//
// Note: the inspector currently has NO tools — it operates as pure LLM
// inspection of the document content provided in its prompt. This is
// intentional for simplicity in 1.6a; in 1.6b we may add specialized tools.
func DocumentInspector() claude.AgentDefinition {
	return claude.AgentDefinition{
		Name: "document_inspector",
		Description: "Inspects a document and returns a technical structural summary: " +
			"approximate page count, primary language, presence of tables or forms, " +
			"document type confidence, and any structural anomalies. Does NOT extract content. " +
			"Use BEFORE deciding how to process a document.",
		System: `You are a document inspector. Your job is to examine the document provided in the user message and produce a CONCISE technical report on its structure.

Required output format (plain text, max 200 words):

PAGES: <approximate count>
LANGUAGE: <primary language code, e.g., en, es, ru>
TYPE: <best guess: invoice, receipt, contract, report, article, form, other>
TYPE_CONFIDENCE: <high|medium|low>
HAS_TABLES: <yes|no>
HAS_FORMS: <yes|no>
ANOMALIES: <comma-separated list, or "none">
NOTES: <one-sentence summary of structural characteristics>

Rules:
- Report only what you can directly observe in the document
- Use "unknown" for any field you cannot determine confidently
- Do NOT extract specific values (amounts, names, dates) — that's not your job
- Keep the entire response under 200 words. Brevity is critical.`,
		AllowedTools:  nil, // no tools — pure inspection
		MaxIterations: 3,   // should finish in 1; 3 is safety margin
	}
}

package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/skyabove/documind/internal/claude"
)

const documentInspectionInputSchema = `{
  "type": "object",
  "properties": {
    "document_type": {
      "type": "string",
      "enum": ["invoice", "receipt", "contract", "report", "form", "letter", "statement", "certificate", "article", "other", "unknown"],
      "description": "Best structural classification based only on provided document context."
    },
    "type_confidence": {
      "type": "string",
      "enum": ["high", "medium", "low"],
      "description": "Confidence in document_type."
    },
    "primary_language": {
      "type": "string",
      "enum": ["en", "es", "ru", "mixed", "unknown"],
      "description": "Primary language visible in the provided context."
    },
    "has_tables": {
      "type": "string",
      "enum": ["yes", "no", "unknown"]
    },
    "has_forms": {
      "type": "string",
      "enum": ["yes", "no", "unknown"]
    },
    "recommended_extraction": {
      "type": "object",
      "properties": {
        "summary": {"type": "boolean"},
        "entities": {"type": "boolean"},
        "money_normalization_expected": {"type": "boolean"}
      },
      "required": ["summary", "entities", "money_normalization_expected"]
    },
    "entity_targets": {
      "type": "array",
      "items": {
        "type": "string",
        "enum": ["money", "dates", "organizations", "people", "addresses", "document_ids", "emails", "phones", "line_items", "signatures", "unknown"]
      },
      "description": "Entity categories the coordinator should pay attention to."
    },
    "risks": {
      "type": "array",
      "items": {
        "type": "string",
        "enum": ["scanned_pdf", "low_text_quality", "ambiguous_currency", "multiple_documents", "conflicting_values", "unknown_document_type", "none"]
      },
      "description": "Processing risks observed or implied by the provided context."
    },
    "notes": {
      "type": "string",
      "description": "One short operational note for the coordinator."
    }
  },
  "required": [
    "document_type",
    "type_confidence",
    "primary_language",
    "has_tables",
    "has_forms",
    "recommended_extraction",
    "entity_targets",
    "risks",
    "notes"
  ]
}`

// RegisterDocumentInspectionTool registers the structured-output tool used only
// by the document_inspector subagent. It records and returns a canonical JSON
// representation of the inspection, so the parent coordinator receives a
// machine-readable Task result instead of a loose prose summary.
func RegisterDocumentInspectionTool(reg *claude.ToolRegistry) error {
	tool := claude.Tool{
		Name: documentInspectionToolName,
		Description: "Record the document inspection result as structured JSON. " +
			"Call exactly once after reading the DOCUMENT_CONTEXT block. Use unknown values rather than guessing.",
		InputSchema: json.RawMessage(documentInspectionInputSchema),
	}

	handler := func(ctx context.Context, input json.RawMessage) (string, error) {
		var payload map[string]any
		if err := json.Unmarshal(input, &payload); err != nil {
			return "", fmt.Errorf("invalid document inspection input: %w", err)
		}

		out, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshal document inspection: %w", err)
		}
		return string(out), nil
	}

	return reg.Register(tool, handler)
}

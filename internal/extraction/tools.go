package extraction

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/skyabove/documind/internal/claude"
)

// Store accumulates tool outputs during an agentic run.
// The tools mutate this struct via their handlers, so after the loop
// finishes we have all extracted data in one place.
//
// This pattern is common when tools function as "structured output slots"
// rather than side-effectful actions.
type Store struct {
	Summary  *Summary
	Entities []Entity
}

// summaryInputSchema is the JSON Schema for extract_document_summary input.
// Design notes (Task Statement 4.3):
//   - All fields required EXCEPT where source docs may genuinely lack info
//   - document_type has "other" fallback to prevent forced miscategorization
//   - main_topics and key_findings are arrays so multiple items are natural
const summaryInputSchema = `{
  "type": "object",
  "properties": {
    "title": {
      "type": "string",
      "description": "The document's title or a concise descriptive title you assign if none exists"
    },
    "document_type": {
      "type": "string",
      "enum": ["invoice", "contract", "article", "report", "receipt", "legal_document", "other"],
      "description": "Best-fit document category. Use 'other' if none apply rather than forcing a misfit."
    },
    "main_topics": {
      "type": "array",
      "items": {"type": "string"},
      "description": "2-5 primary topics discussed in the document. Keep each under 6 words."
    },
    "key_findings": {
      "type": "array",
      "items": {"type": "string"},
      "description": "2-5 specific facts, figures, or conclusions stated in the document. Prefer concrete statements with numbers/dates over generic observations."
    },
    "one_line_abstract": {
      "type": "string",
      "description": "A single sentence (max 30 words) capturing the document's essence."
    }
  },
  "required": ["title", "document_type", "main_topics", "key_findings", "one_line_abstract"]
}`

// entitiesInputSchema is the JSON Schema for extract_key_entities.
const entitiesInputSchema = `{
  "type": "object",
  "properties": {
    "entities": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "type": {
            "type": "string",
            "enum": ["person", "organization", "location", "money", "date", "product", "identifier", "other"],
            "description": "Entity category. Use 'identifier' for IDs like invoice numbers, order IDs, SKUs."
          },
          "value": {
            "type": "string",
            "description": "The exact text as it appears in the document. Do not normalize, translate, or abbreviate."
          },
          "confidence": {
            "type": "string",
            "enum": ["high", "medium", "low"],
            "description": "high: unambiguous explicit mention. medium: implied or partially stated. low: uncertain inference."
          },
          "context": {
            "type": "string",
            "description": "Optional: a 5-15 word snippet showing how the entity is mentioned. Only include when it disambiguates."
          }
        },
        "required": ["type", "value", "confidence"]
      }
    }
  },
  "required": ["entities"]
}`

// RegisterTools installs the two extraction tools into the registry,
// wiring their handlers to mutate the provided Store.
//
// Returning Store via a pointer lets the caller read results after RunAgent
// completes. Each handler parses its tool_use input and populates the
// corresponding Store field.
func RegisterTools(reg *claude.ToolRegistry, store *Store) error {
	err := reg.Register(
		claude.Tool{
			Name: "extract_document_summary",
			Description: "Produce a structured summary of the provided document. " +
				"Call this tool ONCE per document after you have read its content. " +
				"Use this when you need to record the document's high-level overview: " +
				"title, type, main topics, key findings, and a one-line abstract. " +
				"Do not call this tool for entity extraction — use extract_key_entities for that.",
			InputSchema: json.RawMessage(summaryInputSchema),
		},
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var s Summary
			if err := json.Unmarshal(input, &s); err != nil {
				return "", fmt.Errorf("invalid summary input: %w", err)
			}
			store.Summary = &s
			return "Summary recorded successfully.", nil
		},
	)
	if err != nil {
		return err
	}

	err = reg.Register(
		claude.Tool{
			Name: "extract_key_entities",
			Description: "Extract named entities (people, organizations, locations, monetary amounts, dates, product names, identifiers) from the document. " +
				"Call this tool ONCE per document with ALL entities in a single call — do not call multiple times with partial lists. " +
				"Use this when you need structured entity data. " +
				"Do not use this for summarization — use extract_document_summary for overview content. " +
				"Return the entity value as it literally appears in the source, without normalization.",
			InputSchema: json.RawMessage(entitiesInputSchema),
		},
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var el EntityList
			if err := json.Unmarshal(input, &el); err != nil {
				return "", fmt.Errorf("invalid entities input: %w", err)
			}
			store.Entities = el.Entities
			return fmt.Sprintf("Recorded %d entities.", len(el.Entities)), nil
		},
	)
	return err
}

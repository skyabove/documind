package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/skyabove/documind/internal/claude"
)

const documentInspectionInputSchema = `{
  "type": "object",
  "properties": {
    "document_type": {
      "type": "string",
      "enum": [
        "invoice",
        "receipt",
        "bank_transfer_receipt",
        "payment_confirmation",
        "bank_statement",
        "contract",
        "report",
        "form",
        "letter",
        "certificate",
        "article",
        "legal_document",
        "other",
        "unknown"
      ],
      "description": "Best structural/business category for the document. Use 'unknown' when there is not enough evidence; use 'other' only when the type is clear but not listed."
    },
    "type_confidence": {
      "type": "string",
      "enum": ["high", "medium", "low"],
      "description": "Confidence in document_type based only on visible document evidence."
    },
    "primary_language": {
      "type": "string",
      "enum": ["en", "es", "ru", "mixed", "unknown"],
      "description": "Primary language of the visible document text."
    },
    "structure": {
      "type": "string",
      "enum": ["plain_text", "form", "table", "mixed", "unknown"],
      "description": "Dominant structural layout of the document."
    },
    "layout_complexity": {
      "type": "string",
      "enum": ["low", "medium", "high", "unknown"],
      "description": "Low for simple one-page forms/receipts; medium for multi-section layouts; high for dense tables, multiple columns, or mixed embedded structures."
    },
    "text_quality": {
      "type": "string",
      "enum": ["good", "partial", "poor", "unknown"],
      "description": "Quality of visible/extracted text. Use poor for OCR noise, broken ordering, or missing critical fields."
    },
    "approx_pages": {
      "type": "integer",
      "minimum": 1,
      "description": "Approximate number of pages visible or provided."
    },
    "approx_text_volume": {
      "type": "string",
      "enum": ["short", "medium", "long", "unknown"],
      "description": "Short: under ~1 page of text. Medium: several pages. Long: large document or many dense pages."
    },
    "contains_tables": {
      "type": "boolean",
      "description": "Whether the document visibly contains tabular data."
    },
    "contains_form_fields": {
      "type": "boolean",
      "description": "Whether the document is organized as labeled fields, key-value pairs, or a form-like layout."
    },
    "contains_monetary_values": {
      "type": "boolean",
      "description": "Whether the document contains monetary amounts, fees, totals, commissions, or balances."
    },
    "contains_dates": {
      "type": "boolean",
      "description": "Whether the document contains explicit dates."
    },
    "contains_identifiers": {
      "type": "boolean",
      "description": "Whether the document contains account numbers, reference numbers, tax IDs, invoice IDs, order IDs, document IDs, or similar identifiers."
    },
    "contains_parties": {
      "type": "boolean",
      "description": "Whether the document contains named people, organizations, senders, recipients, counterparties, issuers, or beneficiaries."
    },
    "contains_contact_info": {
      "type": "boolean",
      "description": "Whether the document contains phone numbers, email addresses, postal addresses, or contact channels."
    },
    "contains_masked_sensitive_data": {
      "type": "boolean",
      "description": "Whether any sensitive data is masked or partially redacted, such as masked account numbers or hidden ID digits."
    },
    "locale_format": {
      "type": "string",
      "enum": ["en_US", "es_ES", "ru_RU", "mixed", "unknown"],
      "description": "Likely locale convention for numbers and dates. Example: es_ES for decimal comma and DD/MM/YYYY."
    },
    "risks": {
      "type": "array",
      "items": {
        "type": "string",
        "enum": [
          "scanned_pdf",
          "low_text_quality",
          "ambiguous_currency",
          "ambiguous_date_format",
          "masked_identifiers",
          "multi_document_bundle",
          "unknown_document_type"
        ]
      },
      "description": "Processing risks observed from structure or content. Return an empty array when no risks are observed."
    }
  },
  "required": [
    "document_type",
    "type_confidence",
    "primary_language",
    "structure",
    "layout_complexity",
    "text_quality",
    "approx_pages",
    "approx_text_volume",
    "contains_tables",
    "contains_form_fields",
    "contains_monetary_values",
    "contains_dates",
    "contains_identifiers",
    "contains_parties",
    "contains_contact_info",
    "contains_masked_sensitive_data",
    "locale_format",
    "risks"
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

		inspectionJSON := string(out)
		slog.InfoContext(ctx, "document inspection recorded", "inspection", inspectionJSON)
		return inspectionJSON, nil
	}

	return reg.Register(tool, handler)
}

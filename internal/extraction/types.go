// Package extraction defines structured output types and tool definitions
// for document analysis. The tools here use Claude's tool_use mechanism
// to guarantee schema-compliant output (see Task Statement 4.3).
package extraction

// Summary is the structured output of the extract_document_summary tool.
type Summary struct {
	Title           string   `json:"title"`
	DocumentType    string   `json:"document_type"` // invoice, contract, article, report, other
	MainTopics      []string `json:"main_topics"`
	KeyFindings     []string `json:"key_findings"`
	OneLineAbstract string   `json:"one_line_abstract"`
}

// Entity is a single entity extracted from the document.
type Entity struct {
	Type       string `json:"type"`              // person, organization, location, money, date, product, other
	Value      string `json:"value"`             // the literal text as it appears
	Confidence string `json:"confidence"`        // high, medium, low
	Context    string `json:"context,omitempty"` // brief surrounding context, if useful

	// Normalized is added by post-extraction hooks for entities that
	// benefit from canonical representation alongside their literal form.
	// Currently populated for type="money".
	Normalized *NormalizedValue `json:"normalized,omitempty"`
}

// NormalizedValue is a structured, canonical representation of an entity value.
// The shape varies by entity type — for money, we emit Amount + Currency.
// Other entity types may extend this struct with additional fields in the future.
type NormalizedValue struct {
	Amount   *float64 `json:"amount,omitempty"`
	Currency string   `json:"currency,omitempty"`
}

// EntityList is the structured output of the extract_key_entities tool.
type EntityList struct {
	Entities []Entity `json:"entities"`
}

// ExtractionResult is the final composite output returned by our pipeline.
type ExtractionResult struct {
	DocumentID string     `json:"document_id"`
	Summary    *Summary   `json:"summary,omitempty"`
	Entities   []Entity   `json:"entities,omitempty"`
	Iterations int        `json:"iterations"`
	Usage      TokenUsage `json:"usage"`
}

// TokenUsage reports total token consumption for the extraction run.
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

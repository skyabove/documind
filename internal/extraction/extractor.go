package extraction

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/skyabove/documind/internal/claude"
)

// Extractor wraps the agentic extraction flow for a document.
type Extractor struct {
	client *claude.Client
}

// NewExtractor constructs an Extractor bound to a Claude client.
func NewExtractor(client *claude.Client) *Extractor {
	return &Extractor{client: client}
}

// systemPrompt defines the extraction behavior. Notes (Task Statement 4.1):
//   - Explicit instructions about tool-calling order
//   - Explicit instructions about avoiding fabrication
//   - Clear termination criterion (both tools called → end_turn)
const systemPrompt = `You are a document extraction system. Your job is to read the provided document and record structured data using the available tools.

Required workflow:
1. First, call extract_document_summary ONCE to record the document overview.
2. Then, call extract_key_entities ONCE with ALL entities from the document.
3. After both tools have been called, respond with a brief confirmation and stop.

Critical rules:
- Only extract information that is explicitly present in the document. Never fabricate.
- If a field's information is genuinely absent, use an empty list or omit optional fields. Do not invent placeholders.
- For money entities, include the currency symbol or code as it appears (e.g., "$1,250.00", "€500", "USD 15000").
- For dates, preserve the original format from the document. Do not normalize.
- Do not call the same tool twice. If you have already called a tool, use its confirmation as a signal to proceed.`

// Extract runs the agentic extraction pipeline on a PDF.
// Returns a populated ExtractionResult or an error.
func (e *Extractor) Extract(ctx context.Context, documentID string, pdfBytes []byte) (*ExtractionResult, error) {
	registry := claude.NewToolRegistry()
	store := &Store{}
	if err := RegisterTools(registry, store); err != nil {
		return nil, fmt.Errorf("register tools: %w", err)
	}

	// Set up post-tool-use hooks (Task Statement 1.5).
	hooks := claude.NewHookRegistry()
	hooks.AddPostToolUse("extract_key_entities", MoneyNormalizer(store))

	pdfB64 := base64.StdEncoding.EncodeToString(pdfBytes)
	initial := []claude.ContentBlock{
		{
			Type: claude.ContentTypeDocument,
			Source: &claude.DocumentSource{
				Type:      "base64",
				MediaType: "application/pdf",
				Data:      pdfB64,
			},
		},
		{
			Type: claude.ContentTypeText,
			Text: "Extract structured data from this document using the available tools, following the required workflow.",
		},
	}

	result, err := e.client.RunAgent(ctx, claude.RunAgentOptions{
		System:         systemPrompt,
		InitialContent: initial,
		Tools:          registry,
		Hooks:          hooks, // ← NEW
		MaxIterations:  6,
		MaxTokens:      2048,
	})
	if err != nil {
		return nil, fmt.Errorf("agent run: %w", err)
	}

	return &ExtractionResult{
		DocumentID: documentID,
		Summary:    store.Summary,
		Entities:   store.Entities,
		Iterations: result.Iterations,
		Usage: TokenUsage{
			InputTokens:  result.TotalUsage.InputTokens,
			OutputTokens: result.TotalUsage.OutputTokens,
		},
	}, nil
}

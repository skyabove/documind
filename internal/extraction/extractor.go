package extraction

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/skyabove/documind/internal/agents"
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

const systemPrompt = `You are a document extraction coordinator. Your job is to read the provided document and record structured data using the available tools.

Required workflow:
1. First, call the Task tool to invoke the "document_inspector" subagent. Pass it the document context: "Inspect the document I am about to send. Report its structure." (The PDF will be available to the subagent as well — coordinator and subagent both see the document if you reference it.)
2. After the inspector returns, call extract_document_summary ONCE to record the document overview.
3. Then, call extract_key_entities ONCE with ALL entities from the document.
4. After all three calls have completed, respond with a brief confirmation and stop.

Critical rules:
- Only extract information that is explicitly present in the document. Never fabricate.
- For money entities, include the currency symbol or code as it appears.
- For dates, preserve the original format from the document.
- Do not call the same tool twice.`

// Extract runs the agentic extraction pipeline on a PDF.
func (e *Extractor) Extract(ctx context.Context, documentID string, pdfBytes []byte) (*ExtractionResult, error) {
	// --- Tool registry for the COORDINATOR (top-level agent) ---
	coordinatorTools := claude.NewToolRegistry()
	store := &Store{}
	if err := RegisterTools(coordinatorTools, store); err != nil {
		return nil, fmt.Errorf("register extraction tools: %w", err)
	}

	// --- Agent registry for subagents ---
	agentReg := claude.NewAgentRegistry()
	if err := agentReg.Register(agents.DocumentInspector()); err != nil {
		return nil, fmt.Errorf("register inspector: %w", err)
	}

	// --- Shared tools available to subagents (currently empty — inspector has no tools) ---
	sharedTools := claude.NewToolRegistry()

	// --- Wire Task tool into coordinator's registry ---
	if err := claude.RegisterTaskTool(e.client, coordinatorTools, agentReg, sharedTools); err != nil {
		return nil, fmt.Errorf("register Task tool: %w", err)
	}

	// --- Hooks ---
	hooks := claude.NewHookRegistry()
	hooks.AddPostToolUse("extract_key_entities", MoneyNormalizer(store))

	// --- Initial content for coordinator ---
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
			Text: "Extract structured data from this document using the available tools, following the required workflow. Start by inspecting the document with the document_inspector subagent.",
		},
	}

	result, err := e.client.RunAgent(ctx, claude.RunAgentOptions{
		System:         systemPrompt,
		InitialContent: initial,
		Tools:          coordinatorTools,
		Hooks:          hooks,
		MaxIterations:  10, // raised — coordinator now has more steps
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

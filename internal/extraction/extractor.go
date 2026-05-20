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

// systemPrompt configures the coordinator agent's behavior.
//
// The coordinator's job:
//  1. Delegate document inspection to the document_inspector subagent
//  2. Pass explicit document context in the Task prompt
//  3. Use the inspector's structured JSON result as routing guidance
//  4. Record summary/entities with extraction tools
//  5. Wrap up
const systemPrompt = `You are a document extraction coordinator. Your job is to read the attached PDF and record structured data using the available tools.

Required workflow:
1. First, call the Task tool to invoke the "document_inspector" subagent.
2. The Task prompt MUST include a DOCUMENT_CONTEXT block. The subagent cannot see the PDF or this conversation, so include concrete context copied or summarized from what you can observe in the attached PDF:
   - document_id
   - approximate page count if visible, otherwise unknown
   - text_preview with exact visible snippets where possible
   - visible structural signals such as tables, form fields, headings, totals, dates, signatures
3. Read the inspector's returned JSON. Do not ignore it. Use recommended_extraction, entity_targets, and risks to decide what to extract and what to treat cautiously.
4. If recommended_extraction.summary is true or the document type is uncertain, call extract_document_summary ONCE.
5. If recommended_extraction.entities is true or entity_targets is non-empty, call extract_key_entities ONCE with all explicitly present relevant entities.
6. After required extraction tool calls have completed, respond with a brief confirmation and stop.

Critical rules:
- Only extract information that is explicitly present in the document. Never fabricate.
- For money entities, include the currency symbol or code as it appears.
- For dates, preserve the original format from the document.
- Preserve specific document subtype in the summary. For example, do not reduce a bank transfer receipt/payment confirmation to a generic document if the schema gives you a more specific option.
- Prefer semantically exact entity types. Use phone for phone numbers when the entity schema supports it; use identifier for account numbers, references, tax IDs, and document IDs.
- Focus entity extraction on business-relevant content. Avoid extracting generic legal boilerplate, footer/contact lines, or registry locations unless they are directly relevant to the document's purpose.
- If the inspector reports low_text_quality, scanned_pdf, ambiguous_currency, or unknown_document_type, be conservative and use unknown/null-like wording rather than guessing.
- Do not call the same extraction tool twice.`

// Extract runs the agentic extraction pipeline on a PDF.
//
// Architecture:
//   - coordinatorTools: tools available to the top-level agent
//     (Task + extract_document_summary + extract_key_entities)
//   - agentReg: subagent definitions the coordinator may invoke via Task
//   - sharedTools: pool of tools subagents may use (empty for 1.6a —
//     our only subagent has no tools)
//   - hooks: post-tool-use transformations (money normalization)
func (e *Extractor) Extract(ctx context.Context, documentID string, pdfBytes []byte) (*ExtractionResult, error) {
	// --- Coordinator's tool registry ---
	coordinatorTools := claude.NewToolRegistry()
	store := &Store{}
	if err := RegisterTools(coordinatorTools, store); err != nil {
		return nil, fmt.Errorf("register extraction tools: %w", err)
	}

	// --- Subagent definitions ---
	agentReg := claude.NewAgentRegistry()
	if err := agentReg.Register(agents.DocumentInspector()); err != nil {
		return nil, fmt.Errorf("register inspector agent: %w", err)
	}

	// --- Shared tools pool for subagents ---
	// Subagents receive only tools listed in their AgentDefinition.AllowedTools.
	sharedTools := claude.NewToolRegistry()
	if err := agents.RegisterDocumentInspectionTool(sharedTools); err != nil {
		return nil, fmt.Errorf("register document inspection tool: %w", err)
	}

	// --- Wire Task tool into coordinator's registry ---
	if err := claude.RegisterTaskTool(e.client, coordinatorTools, agentReg, sharedTools); err != nil {
		return nil, fmt.Errorf("register Task tool: %w", err)
	}

	// --- PostToolUse hooks (deterministic transformations) ---
	hooks := claude.NewHookRegistry()
	hooks.AddPostToolUse("extract_key_entities", MoneyNormalizer(store))

	// --- Initial content: PDF + instruction ---
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
			Text: fmt.Sprintf(`Extract structured data from this document using the available tools, following the required workflow.

DOCUMENT_METADATA:
- document_id: %s
- pdf_size_bytes: %d

Start by calling Task with agent=document_inspector. In the Task prompt, include a DOCUMENT_CONTEXT block with concrete visible PDF context. Do not send only a generic instruction.`, documentID, len(pdfBytes)),
		},
	}

	// --- Run the coordinator ---
	result, err := e.client.RunAgent(ctx, claude.RunAgentOptions{
		System:         systemPrompt,
		InitialContent: initial,
		Tools:          coordinatorTools,
		Hooks:          hooks,
		MaxIterations:  10,
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

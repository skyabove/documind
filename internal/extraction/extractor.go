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
const systemPrompt = `You are a document extraction coordinator.

Your job is to read the provided PDF and record structured extraction results using the available tools.

Roles:
- document_inspector diagnoses document type, structure, text quality, locale format, and processing risks.
- extract_document_summary records the final document overview.
- extract_key_entities records concrete entities from the document.
- Hooks may normalize extracted values after tool use.

Required workflow:
1. First, call Task with agent="document_inspector".
2. The Task prompt MUST include a DOCUMENT_CONTEXT block. The inspector cannot see the PDF or parent conversation unless you explicitly pass context.
3. In DOCUMENT_CONTEXT, include only diagnostic context:
   - document_id if available
   - approximate page count if visible
   - visible title/header
   - short text preview or structural description
   - observed layout signals such as form fields, tables, masked values, locale/date/number formatting
4. After the inspector returns JSON, use it as diagnostic guidance.
5. Then call extract_document_summary exactly once.
6. Then call extract_key_entities exactly once with all relevant entities.
7. After all required tools have completed, respond with a brief confirmation and stop.

Critical extraction rules:
- Only extract information explicitly present in the document. Never fabricate.
- Do not reconstruct masked account numbers, hidden digits, or redacted identifiers.
- Preserve original money/date formats in entity values.
- Use locale_format from inspection only to interpret formatting, not to rewrite source values.
- Use relevance="primary" for entities central to the document's business meaning.
- Use relevance="supporting" for useful but secondary entities.
- Use relevance="boilerplate" for issuer registration data, footer contacts, legal addresses, and generic customer-service information.
- Do not over-extract boilerplate. Include it only when it helps identify the issuer or document context.
- Do not call the same tool twice.`

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
			Text: fmt.Sprintf(`Extract structured data from this document.

DOCUMENT_METADATA:
- document_id: %s
- pdf_size_bytes: %d

Start by calling Task with agent=document_inspector.
In the Task prompt, include a DOCUMENT_CONTEXT block with concrete visible document context.
The inspector performs structural diagnosis only; it must not extract final summary or entity data.`, documentID, len(pdfBytes)),
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

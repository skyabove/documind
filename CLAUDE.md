# DocuMind

DocuMind is a document extraction and analysis tool. Users upload PDFs, ask questions, and receive structured answers with citations.

## Tech Stack

- **Backend:** Go 1.22+ with standard library `net/http` + `chi` router
- **Frontend:** HTMX + Tailwind CSS (no React, no npm build)
- **Storage:** PostgreSQL for metadata, local filesystem for PDFs (S3-compatible later)
- **LLM:** Anthropic Claude API via direct HTTP (no SDK — this is intentional for learning)
- **MCP:** Custom MCP server in Python, communicates with Go backend via HTTP

## Coding Standards

### Go
- Use standard library wherever possible; justify any third-party dependency
- All exported types and functions must have godoc comments
- Errors are values: use `fmt.Errorf("context: %w", err)` to wrap, never swallow
- No panics in request handlers — return errors to the caller
- Context-first: every function that does I/O takes `context.Context` as first argument

### Project Conventions
- Business logic lives in `internal/`, never in `cmd/`
- HTTP handlers are thin — they parse input, call business logic, format output
- Database queries use parametrized statements only (never string concatenation)
- Configuration via environment variables, loaded at startup in `cmd/server/main.go`

### Testing
- Table-driven tests with descriptive names
- Use `t.Run(name, func(t *testing.T))` subtests
- Integration tests use `_integration_test.go` suffix and require build tag `//go:build integration`

## Anti-Patterns to Avoid

- No global state except the logger
- No `interface{}` / `any` unless strictly necessary — prefer concrete types
- No premature abstraction: write the concrete version first, extract interface only when a second implementation exists

## When Working on This Project

- For bug fixes in single files: direct execution is fine
- For changes affecting 3+ files or architectural decisions: use plan mode
- Before adding a new dependency, confirm there's no standard library solution
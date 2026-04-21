# DocuMind — Project Instructions

## Project Overview
DocuMind is a personal AI agent for working with documents. It uses the Claude API to extract, analyze, and query information from uploaded documents.

## Architecture
- **cmd/server** — application entrypoint
- **internal/api** — HTTP handlers (HTMX-driven UI + JSON API)
- **internal/claude** — Claude API client wrapper
- **internal/documents** — document storage and retrieval
- **internal/extraction** — text extraction from PDF/DOCX/etc.
- **web/templates** — HTMX HTML templates
- **web/static** — CSS, JS, assets

## Tech Stack
- Go 1.22+
- HTMX for frontend interactions
- Claude API (claude-sonnet-4-6 by default)
- Docker / docker-compose for local dev

## Development Guidelines
- Follow rules in `.claude/rules/` for domain-specific guidance
- Use `make` targets for common tasks (see Makefile)
- All handlers must return proper HTTP status codes and structured errors
- Tests live next to the code they test (`_test.go` files)

## Running Locally
```bash
make dev        # run with hot reload
make test       # run all tests
make build      # build binary
docker-compose up  # full stack
```

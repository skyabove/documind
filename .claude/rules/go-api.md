---
paths: ["internal/api/**/*.go", "cmd/server/**/*.go"]
---

# Go API Handler Conventions

## Handler Signature

All HTTP handlers follow this pattern:

    func (s *Server) handleSomething(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        // ...
    }

Never use the package-level `http.HandleFunc` — always methods on `*Server`.

## Request Parsing

- Decode JSON into typed request structs, never `map[string]interface{}`
- Validate input **before** calling business logic
- Use `http.MaxBytesReader` for any endpoint that accepts a body

## Response Formatting

- Success: `writeJSON(w, http.StatusOK, response)`
- Client errors: `writeError(w, http.StatusBadRequest, "human-readable message")`
- Server errors: `writeError(w, http.StatusInternalServerError, "internal error")` + log actual error

Never expose internal error messages to the client.

## Context Propagation

Always pass `r.Context()` to downstream calls. Never create `context.Background()` inside a handler — that breaks cancellation on client disconnect.
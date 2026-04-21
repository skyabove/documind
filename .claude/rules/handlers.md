# HTTP Handler Rules

## Response format
- Success responses: `200 OK` with JSON body `{"data": ...}` or HTMX partial HTML
- Client errors: `4xx` with JSON body `{"error": "human-readable message"}`
- Server errors: `500` with generic message — never leak internal details to client

## Handler structure
```go
func (h *Handler) MethodResource(w http.ResponseWriter, r *http.Request) {
    // 1. Parse & validate input
    // 2. Call service/business logic
    // 3. Render response
}
```
- Handlers must not contain business logic — delegate to service layer
- Keep handlers thin: parse → call → respond

## Input validation
- Validate all user input before processing; return `400` on invalid input
- Use `r.FormValue` / `json.Decoder` — never `r.Form` without `ParseForm`
- Limit request body size with `http.MaxBytesReader`

## HTMX specifics
- Check `HX-Request` header to distinguish HTMX vs full-page requests
- Return HTML partials for HTMX requests; full page otherwise
- Use `HX-Redirect` header for post-action redirects instead of `http.Redirect`

## Middleware
- Auth, logging, and recovery belong in middleware, not handlers
- Use `r.Context()` to pass request-scoped values (user ID, request ID)

## File uploads
- Always validate MIME type server-side (don't trust `Content-Type` header)
- Stream large files — don't buffer entire upload in memory
- Store with a generated UUID name, not the original filename

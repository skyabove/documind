# Go API Rules

## General
- Use `errors.New` / `fmt.Errorf` with `%w` for error wrapping; never discard errors
- Prefer explicit returns over named return variables except for defer-based cleanup
- Keep functions focused: one responsibility, fits in a screen
- Use `context.Context` as the first parameter in all I/O functions

## Naming
- Exported types/functions use PascalCase; unexported use camelCase
- Interfaces end in `-er` (e.g., `DocumentStore`, `Extractor`)
- Avoid stuttering: `documents.Store`, not `documents.DocumentStore`

## Packages
- `internal/` packages are not importable outside the module — use freely for isolation
- No circular imports; dependency direction: `api → documents/extraction/claude`
- Keep `main.go` thin: wire dependencies, start server, handle signals

## Error handling
- Return errors up the call stack; log only at the top (handler level)
- Wrap with context: `fmt.Errorf("extracting text from %s: %w", name, err)`
- Define sentinel errors with `errors.New` for ones callers need to check

## Structs & interfaces
- Define interfaces where they are *used*, not where they are *implemented*
- Use struct embedding sparingly; prefer explicit field promotion

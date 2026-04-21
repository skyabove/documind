# /review — Code Review Command

Perform a thorough code review of the specified file or recent changes.

## What to check

1. **Correctness** — does the logic do what it claims?
2. **Error handling** — are all errors handled explicitly, no silent discards?
3. **Security** — no injection vectors, secrets in code, or unsafe operations
4. **Go idioms** — follows effective Go conventions (named returns, defer, etc.)
5. **HTTP handlers** — proper status codes, structured error responses, no naked `http.Error`
6. **Tests** — adequate coverage for new logic; no mocked DB unless justified
7. **Performance** — no obvious N+1 queries, unbounded allocations, or blocking calls in hot paths

## Output format

For each issue found:
- **File:Line** — short description
- Severity: `critical` / `warning` / `suggestion`
- Recommended fix (inline code snippet if helpful)

If no issues: state "LGTM" with a one-line summary of what was reviewed.

# Test Rules

## Structure
- Test files live alongside source: `foo.go` → `foo_test.go`
- Use `package foo_test` (black-box) by default; `package foo` only when testing unexported internals
- Group related cases with `t.Run("scenario name", ...)` subtests

## Assertions
- Use `testify/assert` and `testify/require`; `require` for fatal preconditions
- One logical assertion per subtest when possible
- Include meaningful failure messages: `assert.Equal(t, want, got, "after processing doc %s", docID)`

## Mocking & fakes
- Do NOT mock the database or storage layer — use real implementations against test data
- Use interfaces to inject fakes for external services (Claude API, file system)
- Fakes live in `internal/<pkg>/testdata/` or a `fake_*.go` file in the same package

## Test data
- Static fixtures go in `testdata/` directories (e.g., sample PDFs, JSON responses)
- Never commit real documents or PII as test fixtures

## Coverage
- Every exported function needs at least one test
- Error paths must be tested, not just happy paths
- Table-driven tests preferred for functions with multiple input/output variants

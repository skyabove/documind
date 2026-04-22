# Code Review

Review the currently open file or the files listed by the user against our standards:

1. **Correctness**: Are there bugs, race conditions, or incorrect error handling?
2. **Conventions**: Does the code follow the rules in `CLAUDE.md` and relevant `.claude/rules/` files?
3. **Testability**: Is the code structured so it can be tested? Are there tests?
4. **Simplicity**: Is there unnecessary abstraction or complexity that could be removed?

For each issue, provide:
- **Location**: file:line
- **Severity**: critical / important / minor
- **Problem**: one sentence describing the issue
- **Fix**: concrete suggestion

Only report real issues. Do not comment on style preferences or add commentary about code that is correct.
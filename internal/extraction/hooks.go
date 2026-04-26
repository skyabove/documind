package extraction

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/skyabove/documind/internal/claude"
)

// MoneyNormalizer creates a PostToolUse hook that adds normalized amounts
// to money entities after extract_key_entities completes.
//
// The hook closes over the same Store the tool handlers populated, so it
// can read store.Entities, parse monetary values deterministically, and
// write back enriched data.
//
// This is Task Statement 1.5 — using hooks for deterministic transformation
// where prompt-based instructions ("normalize amounts to floats") would be
// unreliable: LLMs are inconsistent at format conversion, especially for
// locale-specific number formats.
func MoneyNormalizer(store *Store) claude.PostToolUseHook {
	return func(ctx context.Context, call claude.ToolCall) error {
		// Only act on the entities tool — other tools are unaffected.
		if call.Name != "extract_key_entities" {
			return nil
		}
		if call.IsError {
			return nil
		}

		for i := range store.Entities {
			e := &store.Entities[i]
			if e.Type != "money" {
				continue
			}
			amount, currency, ok := parseMoney(e.Value)
			if !ok {
				// Couldn't parse — leave entity unchanged. Better than
				// guessing and producing wrong normalized values.
				continue
			}
			e.Normalized = &NormalizedValue{
				Amount:   &amount,
				Currency: currency,
			}
		}
		return nil
	}
}

// moneyPattern matches a numeric portion followed by an optional currency.
// It accepts both "1,234.56" (US) and "1.234,56" (EU) formats and common
// currency symbols/codes.
var moneyPattern = regexp.MustCompile(
	`(?i)([\d.,]+)\s*(EUR|USD|GBP|RUB|JPY|CHF|€|\$|£|¥|₽)?` +
		`|(EUR|USD|GBP|RUB|JPY|CHF|€|\$|£|¥|₽)\s*([\d.,]+)`,
)

// currencyMap normalizes currency symbols to ISO codes.
var currencyMap = map[string]string{
	"€":   "EUR",
	"$":   "USD",
	"£":   "GBP",
	"¥":   "JPY",
	"₽":   "RUB",
	"EUR": "EUR",
	"USD": "USD",
	"GBP": "GBP",
	"RUB": "RUB",
	"JPY": "JPY",
	"CHF": "CHF",
}

// parseMoney extracts a numeric amount and ISO currency code from a string.
// Returns ok=false if the string doesn't match expected patterns.
func parseMoney(s string) (amount float64, currency string, ok bool) {
	s = strings.TrimSpace(s)
	matches := moneyPattern.FindStringSubmatch(s)
	if matches == nil {
		return 0, "", false
	}

	// The regex has two alternative groups; figure out which matched.
	var numericPart, currencyPart string
	if matches[1] != "" {
		numericPart = matches[1]
		currencyPart = matches[2]
	} else {
		currencyPart = matches[3]
		numericPart = matches[4]
	}

	if numericPart == "" {
		return 0, "", false
	}

	amount, err := parseNumeric(numericPart)
	if err != nil {
		return 0, "", false
	}

	currency = currencyMap[strings.ToUpper(currencyPart)]
	return amount, currency, true
}

// parseNumeric handles both "1,234.56" (US) and "1.234,56" (EU) formats.
// Heuristic: if the string contains both '.' and ',', the rightmost one
// is the decimal separator. If only one, ambiguity is resolved by counting
// digits after it (1-2 digits → decimal, 3+ → thousands).
func parseNumeric(s string) (float64, error) {
	s = strings.TrimSpace(s)
	hasDot := strings.Contains(s, ".")
	hasComma := strings.Contains(s, ",")

	switch {
	case hasDot && hasComma:
		lastDot := strings.LastIndex(s, ".")
		lastComma := strings.LastIndex(s, ",")
		if lastComma > lastDot {
			// EU format: "1.234,56" — comma is decimal
			s = strings.ReplaceAll(s, ".", "")
			s = strings.Replace(s, ",", ".", 1)
		} else {
			// US format: "1,234.56" — dot is decimal
			s = strings.ReplaceAll(s, ",", "")
		}
	case hasDot && !hasComma:
		// Could be "1.234" (EU thousands) or "1.50" (decimal).
		// Heuristic on digits after the dot.
		idx := strings.LastIndex(s, ".")
		afterDot := len(s) - idx - 1
		if afterDot == 3 {
			// likely thousands separator
			s = strings.ReplaceAll(s, ".", "")
		}
		// else assume decimal — already valid Go float syntax
	case !hasDot && hasComma:
		// Could be "1,234" (US thousands) or "1,50" (EU decimal).
		idx := strings.LastIndex(s, ",")
		afterComma := len(s) - idx - 1
		if afterComma == 3 {
			s = strings.ReplaceAll(s, ",", "")
		} else {
			s = strings.Replace(s, ",", ".", 1)
		}
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %q as float: %w", s, err)
	}
	return f, nil
}

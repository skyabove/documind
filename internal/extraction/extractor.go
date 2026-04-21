package extraction

import (
	"context"
	"fmt"
	"strings"
)

// Extractor extracts plain text from raw document bytes.
type Extractor interface {
	Extract(ctx context.Context, contentType string, data []byte) (string, error)
}

// PlainTextExtractor handles text/plain documents.
type PlainTextExtractor struct{}

func (PlainTextExtractor) Extract(_ context.Context, contentType string, data []byte) (string, error) {
	if !strings.HasPrefix(contentType, "text/") {
		return "", fmt.Errorf("unsupported content type: %s", contentType)
	}
	return string(data), nil
}

// Multi dispatches to registered extractors by content type prefix.
type Multi struct {
	extractors map[string]Extractor
}

func NewMulti() *Multi {
	m := &Multi{extractors: make(map[string]Extractor)}
	m.Register("text/", PlainTextExtractor{})
	return m
}

func (m *Multi) Register(prefix string, e Extractor) {
	m.extractors[prefix] = e
}

func (m *Multi) Extract(ctx context.Context, contentType string, data []byte) (string, error) {
	for prefix, e := range m.extractors {
		if strings.HasPrefix(contentType, prefix) {
			return e.Extract(ctx, contentType, data)
		}
	}
	return "", fmt.Errorf("no extractor registered for content type: %s", contentType)
}

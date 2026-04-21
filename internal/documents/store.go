package documents

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrNotFound is returned when a document does not exist.
var ErrNotFound = errors.New("document not found")

// Document represents a stored document.
type Document struct {
	ID          string
	Name        string
	ContentType string
	Content     []byte
	Text        string // extracted plain text
	CreatedAt   time.Time
}

// Store defines the document persistence interface.
type Store interface {
	Save(ctx context.Context, doc *Document) error
	Get(ctx context.Context, id string) (*Document, error)
	List(ctx context.Context) ([]*Document, error)
	Delete(ctx context.Context, id string) error
}

// MemoryStore is an in-memory Store implementation for development.
type MemoryStore struct {
	mu   sync.RWMutex
	docs map[string]*Document
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{docs: make(map[string]*Document)}
}

func (s *MemoryStore) Save(_ context.Context, doc *Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[doc.ID] = doc
	return nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (*Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.docs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return doc, nil
}

func (s *MemoryStore) List(_ context.Context) ([]*Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Document, 0, len(s.docs))
	for _, d := range s.docs {
		out = append(out, d)
	}
	return out, nil
}

func (s *MemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.docs[id]; !ok {
		return ErrNotFound
	}
	delete(s.docs, id)
	return nil
}

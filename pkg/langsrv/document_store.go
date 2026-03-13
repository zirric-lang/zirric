package langsrv

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type documentSnapshot struct {
	Text    string
	Version int32
	Dirty   bool
	ModTime time.Time
}

type document struct {
	path    string
	version int32
	text    string
	dirty   bool
	modTime time.Time
}

type documentStore struct {
	mu   sync.RWMutex
	docs map[string]*document
}

func newDocumentStore() *documentStore {
	return &documentStore{docs: make(map[string]*document)}
}

func (s *documentStore) Open(path string, version int32, text string) {
	path = cleanPath(path)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[path] = &document{
		path:    path,
		version: version,
		text:    text,
		dirty:   true,
		modTime: time.Now(),
	}
}

func (s *documentStore) ApplyChanges(path string, version int32, changes []protocol.TextDocumentContentChangeEvent) error {
	path = cleanPath(path)
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, ok := s.docs[path]
	if !ok {
		return fmt.Errorf("document not open: %s", path)
	}
	text, err := applyContentChanges(doc.text, changes)
	if err != nil {
		return err
	}
	doc.text = text
	doc.version = version
	doc.dirty = true
	doc.modTime = time.Now()
	return nil
}

func (s *documentStore) Save(path string, text *string) {
	path = cleanPath(path)
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, ok := s.docs[path]
	if !ok {
		return
	}
	if text != nil {
		doc.text = *text
	}
	doc.dirty = false
	doc.modTime = time.Now()
}

func (s *documentStore) Close(path string) {
	path = cleanPath(path)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.docs, path)
}

func (s *documentStore) Snapshot(path string) (documentSnapshot, bool) {
	path = cleanPath(path)
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.docs[path]
	if !ok {
		return documentSnapshot{}, false
	}
	return documentSnapshot{
		Text:    doc.text,
		Version: doc.version,
		Dirty:   doc.dirty,
		ModTime: doc.modTime,
	}, true
}

func (s *documentStore) Paths() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	paths := make([]string, 0, len(s.docs))
	for path := range s.docs {
		paths = append(paths, path)
	}
	return paths
}

func cleanPath(path string) string {
	if path == "" {
		return path
	}
	return filepath.Clean(path)
}

func applyContentChanges(text string, changes []protocol.TextDocumentContentChangeEvent) (string, error) {
	for _, change := range changes {
		if change.Range == nil {
			text = change.Text
			continue
		}
		start, end := change.Range.IndexesIn(text)
		if isRangeIndexInvalid(*change.Range, start, end, text) {
			return "", fmt.Errorf("invalid text change range")
		}
		text = text[:start] + change.Text + text[end:]
	}
	return text, nil
}

func isRangeIndexInvalid(r protocol.Range, start int, end int, text string) bool {
	if start > end || start < 0 || end < 0 || start > len(text) || end > len(text) {
		return true
	}
	if start == 0 && (r.Start.Line > 0 || r.Start.Character > 0) {
		return true
	}
	if end == 0 && (r.End.Line > 0 || r.End.Character > 0) {
		return true
	}
	return false
}

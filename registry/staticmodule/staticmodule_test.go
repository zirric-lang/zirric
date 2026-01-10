package staticmodule

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/registry"
)

func TestStaticModuleSources(t *testing.T) {
	src := NewSourceString(registry.LogicalURI("src/foo"), "bar")
	mod := NewModule(registry.LogicalURI("mod"), []registry.Source{src})

	if mod.URI() != registry.LogicalURI("mod") {
		t.Fatalf("expected URI mod, got %s", mod.URI())
	}

	sources, err := mod.Sources()
	if err != nil {
		t.Fatalf("Sources returned error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	data, err := sources[0].Read()
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if string(data) != "bar" {
		t.Fatalf("expected source content bar, got %s", string(data))
	}
}

func TestNewSource(t *testing.T) {
	s := NewSource(registry.LogicalURI("foo"), []byte("baz"))
	if s.URI() != registry.LogicalURI("foo") {
		t.Fatalf("expected URI foo, got %s", s.URI())
	}
	data, err := s.Read()
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if string(data) != "baz" {
		t.Fatalf("expected content baz, got %s", string(data))
	}
}

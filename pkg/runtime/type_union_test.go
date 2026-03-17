package runtime

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

func makeTestSymbol(name string, constantId int) *ast.Symbol {
	ident := ast.MakeIdentifier(token.Token{Literal: name})
	decl := ast.MakeDeclUnion(token.Token{}, ident)
	typeSym := &ast.Symbol{
		Name:       name,
		ConstantId: &constantId,
	}
	return &ast.Symbol{
		Name:       name,
		Decl:       decl,
		TypeSymbol: typeSym,
	}
}

func TestUnionTypeIsMember(t *testing.T) {
	sym := makeTestSymbol("TestUnion", 100)
	members := []TypeId{1, 2, 3}
	ut := MakeUnionType(sym, members)

	if !ut.IsMember(1) {
		t.Error("expected type id 1 to be a member")
	}
	if !ut.IsMember(2) {
		t.Error("expected type id 2 to be a member")
	}
	if !ut.IsMember(3) {
		t.Error("expected type id 3 to be a member")
	}
	if ut.IsMember(4) {
		t.Error("expected type id 4 NOT to be a member")
	}
	if ut.IsMember(0) {
		t.Error("expected type id 0 NOT to be a member")
	}
}

func TestUnionTypeIsMemberEmpty(t *testing.T) {
	sym := makeTestSymbol("EmptyUnion", 200)
	ut := MakeUnionType(sym, nil)

	if ut.IsMember(1) {
		t.Error("empty union should have no members")
	}
}

func TestUnionTypeInspect(t *testing.T) {
	sym := makeTestSymbol("Number", 100)
	ut := MakeUnionType(sym, []TypeId{1, 2})

	got := ut.Inspect()
	if got != "union Number" {
		t.Fatalf("expected 'union Number', got %q", got)
	}
}

func TestUnionTypeLookup(t *testing.T) {
	sym := makeTestSymbol("Number", 100)
	ut := MakeUnionType(sym, []TypeId{1, 2})

	if ut.Lookup("anything") != nil {
		t.Error("Lookup should return nil")
	}
}

func TestUnionTypeConstantId(t *testing.T) {
	constId := 42
	sym := makeTestSymbol("Number", constId)
	ut := MakeUnionType(sym, []TypeId{1, 2})

	if ut.TypeConstantId() != TypeId(constId) {
		t.Fatalf("expected TypeConstantId %d, got %d", constId, ut.TypeConstantId())
	}
}

func TestUnionTypeAttributes(t *testing.T) {
	sym := makeTestSymbol("Number", 100)
	ut := MakeUnionType(sym, []TypeId{1, 2})

	if ut.Attributes != nil {
		t.Fatal("expected nil attributes initially")
	}

	ut.Attributes = map[TypeId]int{TypeId(5): 10}
	if len(ut.Attributes) != 1 {
		t.Fatalf("expected 1 attribute, got %d", len(ut.Attributes))
	}
	if ut.Attributes[TypeId(5)] != 10 {
		t.Fatalf("expected attribute value 10, got %d", ut.Attributes[TypeId(5)])
	}
}

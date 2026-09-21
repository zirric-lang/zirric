package debuginfo_test

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/debuginfo"
	"code.knabel.dev/zirric-lang/zirric/pkg/op"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

func TestLookupFindsTheInstructionCoveringAnIP(t *testing.T) {
	instructions := op.Instructions{1, 2, 3, 4, 5, 6, 7, 8}
	first := token.MakeSource("main.zirr", 0, 1, 1)
	second := token.MakeSource("main.zirr", 0, 4, 2)

	table := debuginfo.NewTable()
	table.Attach(instructions, "greet", []debuginfo.Entry{
		{Offset: 0, Source: first},
		{Offset: 4, Source: second},
	})

	// One instruction spans several bytes, so an ip inside it belongs to the entry at or before it.
	for _, tt := range []struct {
		ip   int
		want *token.Source
	}{
		{0, first}, {1, first}, {3, first}, {4, second}, {7, second},
	} {
		got, ok := table.Lookup(instructions, tt.ip)
		if !ok {
			t.Fatalf("ip %d: expected a position", tt.ip)
		}
		if got.Source != tt.want {
			t.Errorf("ip %d: expected %v, got %v", tt.ip, tt.want, got.Source)
		}
		if got.Function != "greet" {
			t.Errorf("ip %d: expected the function name, got %q", tt.ip, got.Function)
		}
	}
}

func TestLookupDeclinesAStreamItNeverSaw(t *testing.T) {
	table := debuginfo.NewTable()
	if _, ok := table.Lookup(op.Instructions{1, 2, 3}, 0); ok {
		t.Error("expected no position for an unrecorded stream")
	}
	if _, ok := table.Lookup(nil, 0); ok {
		t.Error("expected no position for no instructions")
	}
}

func TestLookupDeclinesAStreamOfADifferentLength(t *testing.T) {
	// A slice sharing its start with a recorded one is not that stream, and answering for it would point at the wrong source.
	instructions := op.Instructions{1, 2, 3, 4, 5, 6, 7, 8}
	table := debuginfo.NewTable()
	table.Attach(instructions, "greet", []debuginfo.Entry{{Offset: 0, Source: token.MakeSource("main.zirr", 0, 1, 1)}})

	if _, ok := table.Lookup(instructions[:4], 0); ok {
		t.Error("expected no position for a stream of a different length")
	}
}

func TestAttachIgnoresNothingToRecord(t *testing.T) {
	table := debuginfo.NewTable()
	table.Attach(nil, "x", []debuginfo.Entry{{Offset: 0}})
	table.Attach(op.Instructions{1, 2}, "x", nil)
	if _, ok := table.Lookup(op.Instructions{1, 2}, 0); ok {
		t.Error("expected nothing to be recorded")
	}
}

func TestEntriesAreFoundWhateverOrderTheyWereGivenIn(t *testing.T) {
	instructions := op.Instructions{1, 2, 3, 4}
	early := token.MakeSource("main.zirr", 0, 1, 1)
	late := token.MakeSource("main.zirr", 0, 9, 1)

	table := debuginfo.NewTable()
	table.Attach(instructions, "f", []debuginfo.Entry{{Offset: 2, Source: late}, {Offset: 0, Source: early}})

	got, _ := table.Lookup(instructions, 1)
	if got.Source != early {
		t.Errorf("expected the earlier entry, got %v", got.Source)
	}
	got, _ = table.Lookup(instructions, 3)
	if got.Source != late {
		t.Errorf("expected the later entry, got %v", got.Source)
	}
}

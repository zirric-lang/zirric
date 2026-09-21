// Package debuginfo maps compiled instructions back to the source they came from.
//
// It exists so that a crash can be explained without every value, frame and function having to carry its own position around for the whole run. The mapping lives beside the program, is consulted only once something has already gone wrong, and costs nothing while a program is working.
package debuginfo

import (
	"reflect"
	"sort"

	"code.knabel.dev/zirric-lang/zirric/pkg/op"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// Table holds, for each stream of instructions, where each of them came from.
type Table struct {
	streams map[uintptr]*stream
}

type stream struct {
	// name is what to call a frame running these instructions, e.g. a function's own name.
	name string
	// length guards against a stale key: a stream that has been replaced by another at the same address would otherwise answer for it.
	length int
	// entries are sorted by offset, so the instruction covering an ip is the last one at or before it.
	entries []Entry
}

// Entry records that the instruction at an offset came from a place in the source.
type Entry struct {
	Offset int
	Source *token.Source
}

// Position is where a frame is, and what to call it.
type Position struct {
	Function string
	Source   *token.Source
}

func NewTable() *Table {
	return &Table{streams: map[uintptr]*stream{}}
}

// Attach records where the instructions of a finished stream came from.
//
// It must only be called once the stream is complete: a slice that is still being appended to moves as it grows, and the address it is keyed by would no longer be the one a frame runs.
func (t *Table) Attach(instructions op.Instructions, name string, entries []Entry) {
	if t == nil || len(instructions) == 0 || len(entries) == 0 {
		return
	}
	sorted := make([]Entry, len(entries))
	copy(sorted, entries)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Offset < sorted[j].Offset })
	t.streams[keyOf(instructions)] = &stream{name: name, length: len(instructions), entries: sorted}
}

// Lookup returns where the instruction at ip came from, or false when this stream was never recorded.
func (t *Table) Lookup(instructions op.Instructions, ip int) (Position, bool) {
	if t == nil || len(instructions) == 0 {
		return Position{}, false
	}
	found, ok := t.streams[keyOf(instructions)]
	if !ok || found.length != len(instructions) {
		return Position{}, false
	}
	// The instruction covering ip is the last one recorded at or before it, since one instruction spans several bytes.
	at := sort.Search(len(found.entries), func(i int) bool { return found.entries[i].Offset > ip })
	if at == 0 {
		return Position{Function: found.name}, true
	}
	return Position{Function: found.name, Source: found.entries[at-1].Source}, true
}

// keyOf identifies a stream by where its bytes live.
//
// Each stream is compiled into a slice of its own and kept alive by the program, so its address is a stable name for it — and one that costs nothing to carry, which naming it any other way would not.
func keyOf(instructions op.Instructions) uintptr {
	return reflect.ValueOf(instructions).Pointer()
}

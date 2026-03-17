package op

const (
	_ Opcode = iota
	Const
	ConstVoid
	ConstTrue
	ConstFalse
	Pop

	Array
	Dict
	Module

	GetIndex
	GetField
	SetField
	SetIndex
	Len
	ArrayAppend

	// does not consume, just assert top value's type
	AssertType

	Jump
	JumpTrue
	JumpFalse

	Negate
	Invert

	Add
	Sub
	Mul
	Div
	Mod

	Equal
	NotEqual
	GreaterThan
	GreaterThanOrEqual
	LessThan
	LessThanOrEqual

	MakeAttribute
	Call
	Return
	GetGlobal
	SetGlobal
	GetLocal
	SetLocal

	// Serves as instruction to optionally pause on breakpoints.
	// Will not be compiled for non debugging sessions.
	Debug

	// Closure-related opcodes

	// MakeClosure creates a *Closure from the CompiledFunction at
	// constants[const_id]. It pops free_count values from the stack
	// (pushed in declaration order) and stores them in the Closure.Free slice.
	MakeClosure
	// GetFree pushes closure.Free[index] onto the stack without unwrapping.
	// Used to forward captured values when building nested closures, or to
	// read const captures directly.
	GetFree
	// GetFreeCell pushes closure.Free[index].(*UpvalueCell).Value.
	// Used to read a captured var binding inside a closure.
	GetFreeCell
	// SetFreeCell pops a value and stores it into
	// closure.Free[index].(*UpvalueCell).Value.
	// Used to write a captured var binding inside a closure.
	SetFreeCell
	// GetLocalCell pushes locals[index].(*UpvalueCell).Value.
	// Used in the enclosing scope to read a var that is captured by a closure.
	GetLocalCell
	// SetLocalCell pops a value and stores it into
	// locals[index].(*UpvalueCell).Value.
	// Used in the enclosing scope to write a var that is captured by a closure.
	SetLocalCell
	// WrapLocal wraps locals[index] in an *UpvalueCell in place:
	// locals[index] = &UpvalueCell{Value: locals[index]}.
	// Emitted right after the initial SetLocal for a captured var.
	WrapLocal
)

var definitions = map[Opcode]*Definition{
	Const:      {"const", []int{2}}, // const id
	ConstVoid:  {"constvoid", []int{}},
	ConstTrue:  {"consttrue", []int{}},
	ConstFalse: {"constfalse", []int{}},
	Pop:        {"pop", []int{}},

	Array:  {"array", []int{}},
	Dict:   {"dict", []int{}},
	Module: {"module", []int{2}},

	GetIndex:    {"getindex", []int{}},
	GetField:    {"getfield", []int{2}}, // name id
	SetField:    {"setfield", []int{2}}, // name id; stack: [..., val, obj] — pops obj then val
	SetIndex:    {"setindex", []int{}},  // stack: [..., val, target, index] — pops index, target, val
	Len:         {"len", []int{}},
	ArrayAppend: {"arrayappend", []int{}},

	AssertType: {"asserttype", []int{2}}, // type id

	Jump:      {"jump", []int{2}},      // address
	JumpTrue:  {"jumptrue", []int{2}},  // address
	JumpFalse: {"jumpfalse", []int{2}}, // address

	Negate: {"negate", []int{}},
	Invert: {"invert", []int{}},

	Add: {"add", []int{}},
	Sub: {"sub", []int{}},
	Mul: {"mul", []int{}},
	Div: {"div", []int{}},
	Mod: {"mod", []int{}},

	Equal:              {"eq", []int{}},
	NotEqual:           {"neq", []int{}},
	GreaterThan:        {"gt", []int{}},
	GreaterThanOrEqual: {"gte", []int{}},
	LessThan:           {"lt", []int{}},
	LessThanOrEqual:    {"lte", []int{}},

	MakeAttribute: {"makeannotation", []int{2}}, // arg count

	Call:      {"call", []int{2}}, // arg count
	Return:    {"return", []int{}},
	GetGlobal: {"getglobal", []int{2}},
	SetGlobal: {"setglobal", []int{2}},
	GetLocal:  {"getlocal", []int{2}},
	SetLocal:  {"setlocal", []int{2}},

	Debug: {"debug", []int{}},

	MakeClosure:  {"makeclosure", []int{2, 2}}, // const id, free count
	GetFree:      {"getfree", []int{2}},        // free index
	GetFreeCell:  {"getfreecell", []int{2}},    // free index
	SetFreeCell:  {"setfreecell", []int{2}},    // free index
	GetLocalCell: {"getlocalcell", []int{2}},   // local index
	SetLocalCell: {"setlocalcell", []int{2}},   // local index
	WrapLocal:    {"wraplocal", []int{2}},      // local index
}

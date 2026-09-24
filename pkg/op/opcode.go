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

	// MakeClosure creates a *Closure from the CompiledFunction at constants[const_id]. It pops free_count values from the stack (pushed in declaration order) and stores them in the Closure.Free slice.
	MakeClosure
	// GetFree pushes closure.Free[index] onto the stack without unwrapping.
	// Used to forward captured values when building nested closures, or to read const captures directly.
	GetFree
	// GetFreeCell pushes closure.Free[index].(*UpvalueCell).Value.
	// Used to read a captured var binding inside a closure.
	GetFreeCell
	// SetFreeCell pops a value and stores it into closure.Free[index].(*UpvalueCell).Value.
	// Used to write a captured var binding inside a closure.
	SetFreeCell
	// GetLocalCell pushes locals[index].(*UpvalueCell).Value.
	// Used in the enclosing scope to read a var that is captured by a closure.
	GetLocalCell
	// SetLocalCell pops a value and stores it into locals[index].(*UpvalueCell).Value.
	// Used in the enclosing scope to write a var that is captured by a closure.
	SetLocalCell
	// WrapLocal wraps locals[index] in an *UpvalueCell in place:
	// locals[index] = &UpvalueCell{Value: locals[index]}.
	// Emitted right after the initial SetLocal for a captured var.
	WrapLocal

	// IsType checks whether the value on top of the stack is of a given type.
	// If the constant is a UnionType, checks membership. Otherwise compares TypeConstantId directly. Pops the value and pushes a Bool result.
	IsType

	// AsOption replaces the value on top of the stack with the Option standing for it: a Some or a None is already one, and anything else is asked for one through the toOption of its @AnyOption.
	// A value carrying neither is a failure, which is what makes `?.` and `??` refuse a value that stands for no option at all.
	AsOption
	// AsResult is AsOption for the Result side, going through the toResult of an @AnyResult.
	AsResult

	// JumpIsType jumps when the value on top of the stack matches the type at the given constant, using the same matching as IsType.
	// It peeks rather than pops, so the value is still there on both paths: that is what lets `?.` leave the None it short-circuited on as the chain's result.
	JumpIsType
	// ReturnIsType returns the value on top of the stack from the current frame when it matches the type at the given constant, and does nothing otherwise.
	// This is how `!.` propagates an error without a jump around the rest of the chain.
	ReturnIsType
	// WrapOption wraps the value on top of the stack in Some unless it already is an Option, which is what makes the result of a `?.` chain an Option either way.
	WrapOption

	// MakeIterYield builds the native `yield` callable for a generic `for <-` dispatch (ZE-010).
	MakeIterYield
	// CallIterate reentrantly calls the `iterate` function (top of stack) with 2 args, unlike Call.
	CallIterate
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

	MakeAttribute: {"makeattribute", []int{2}}, // arg count

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
	IsType:       {"istype", []int{2}},         // const id of type or union

	AsOption: {"asoption", []int{2, 2}}, // const id of the Option union, const id of the AnyOption attribute
	AsResult: {"asresult", []int{2, 2}}, // const id of the Result union, const id of the AnyResult attribute

	JumpIsType:   {"jumpistype", []int{2, 2}}, // address, const id of type or union
	ReturnIsType: {"returnistype", []int{2}},  // const id of type or union
	WrapOption:   {"wrapoption", []int{2, 2}}, // const id of the Option union, const id of the Some data type

	MakeIterYield: {"makeiteryield", []int{2, 2, 2}}, // binding local, body start ip, body end ip
	CallIterate:   {"calliterate", []int{2}},         // arg count (always 2: value, yield)
}

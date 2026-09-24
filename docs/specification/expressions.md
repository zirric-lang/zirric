# Expressions

Zirric is expression-first: most constructs produce a value. This page covers all expression and statement forms, their evaluation semantics, and how control flow works.

For the grammar of each form, see [Syntax § Expressions](/specification/syntax#expressions) and [Syntax § Statements](/specification/syntax#statements).

Examples that write output assume `import fmt` and a writer named `out`, as `const out = os.stdout()` gives. There is no bare `print`: writing goes to a writer you name. See [`fmt`](/stdlib/fmt) and [`os`](/stdlib/os).

## Literals

Zirric supports the following literal forms. Each produces a value of the corresponding built-in type.

```zirric
42 // Int
3.14 // Float
0x8899aa // Int (hexadecimal)
0777 // Int (octal — a leading zero)
0b101010 // Int (binary)
1e10 // Float (scientific)
true // Bool
false // Bool
"Hello, World!" // String
[1, 2, 3] // Array
["key": "value"] // Dict
```

**Array literals** create a new `Array` value. Elements are evaluated left to right.

**Dict literals** create a new `Dict` value. Keys and values are evaluated left to right. Keys must be primitive values — `String`, `Int`, `Float`, `Bool`, `Char` or `Byte`; `String` is by far the most common. An `Array` or a `Dict` cannot be a key.

See [Type System](/specification/typesystem) for details on each type.

## Operators

Binary and unary operators follow the precedence table in [Syntax § Operator Precedence](/specification/syntax#operator-precedence).

### Arithmetic

`+`, `-`, `*`, `/`, `%` operate on `Int` and `Float` values. `+` also concatenates strings. Mixing `Int` and `Float` in arithmetic promotes the result to `Float`. Division of two `Int` values produces an `Int` (truncated).

### Comparison

`==` and `!=` compare any two values for equality. Two values are equal if they have the same type and the same content. `data` instances are compared field by field.

`<`, `<=`, `>`, `>=` compare `Int` and `Float` values. Comparing incompatible types is a runtime error.

Comparison operators are non-associative: `a == b == c` is a syntax error.

### Logical

`&&` (and) and `||` (or) short-circuit: the right operand is only evaluated if needed. Both operands must be `Bool`. The result is a `Bool`.

### Prefix

`-x` negates a numeric value. `!x` inverts a `Bool`.

### Fallback

`??` and `!!` stand in for a value that is not there. Both short-circuit: the right operand is only evaluated when the left one turns out to be missing.

Each reads its left operand the same way the guarded accesses do — `??` as an `Option` and `!!` as a `Result` — and then answers with the right operand when what comes back is a `None` or an `Err`, and with what the `Some` or `Ok` holds otherwise. A left operand standing for neither an option nor a result is a runtime error rather than a value that passes through.

```zirric
const city = findUser(42)?.address?.city ?? "Unknown"
const text = readFile("config.txt") !! "default config"
```

Both are right-associative, so `a ?? b ?? c` reads as `a ?? (b ?? c)`. They bind looser than arithmetic and tighter than comparison: `a ?? b + 1` is `a ?? (b + 1)`, and `a ?? b == c` is `(a ?? b) == c`.

## Calls and Member Access

### Function calls

A call evaluates the callee expression, then evaluates all arguments left to right, then invokes the function. If the callee is not a `Func`, this is a runtime error.

```zirric
greet("World")
add(1, 2)
```

Constructor calls for `data` types follow the same syntax — the type name is called as a function with positional arguments matching the field order.

```zirric
const p = Person("Alice", 30)
```

### Member access

The `.` operator accesses a field of a `data` instance, `extern type`, or module.

```zirric
p.name // field of a data instance
strings.join // member of a module
```

Accessing a nonexistent field is a runtime error.

### Guarded member access

`?.` and `!.` read a field the same way `.` does, but only once the value they sit on turns out to be readable — and they read it off what that value _holds_, so no unwrapping step is written.

Both begin by reading the target as the prelude [`Option`](/stdlib/prelude#option) or [`Result`](/stdlib/prelude#result) standing for it. A `Some`, `None`, `Ok` or `Err` is already one; anything else is asked for one through [`@AnyOption`](/stdlib/prelude#anyoption) or [`@AnyResult`](/stdlib/prelude#anyresult), whose callback says how to make one of it, and a value carrying neither is a runtime error. That is what lets a union of your own be read exactly as the prelude's is.

Both bind exactly as tightly as `.`, so they mix freely in one chain.

#### Optional chaining (`?.`)

`target?.field` reads `field` off what `target` holds, unless `target` is absent, in which case the whole surrounding chain is `None` and nothing further in it is evaluated — including an index expression written after it.

A chain that contains a `?.` always produces an `Option`: its result is wrapped in `Some` unless it already is a `Some` or a `None`, so an `Option`-valued field is not wrapped twice. [`??`](#fallback) reads the value back out.

```zirric
const city = user?.address?.city
// Some(city) when both are present, None as soon as either is absent
```

The chain runs to its outermost member or index access. A call ends it: a call whose callee contains a `?.` is a compile error, because the chain's result is an `Option` and an `Option` cannot be called.

#### Result unwrapping (`!.`)

`target!.field` reads `field` off what `target` holds, unless `target` is an `Err`, in which case the enclosing function returns that `Err` and the rest of its body does not run.

```zirric
fn hostOf(result: Config!) -> String! {
	const host = result!.host
	return Ok("host is " + host)
}
// hostOf(Ok(Config("example.com"))) is Ok("host is example.com")
// hostOf(Err("missing file")) is Err("missing file")
```

Because `!.` returns from the function it sits in, it is only allowed inside one: using it in a top-level statement or a declaration's initializer is a compile error.

### Index access

The `[]` operator accesses elements by index (`Array`, `String`, `Binary`) or by key (`Dict`). Indexing a `String` or a `Binary` produces a `Byte`; iterating a `String` yields `Char` values instead, and `chars()` collects those.

```zirric
const arr = [10, 20, 30]
arr[0] // 10

const dict = ["a": 1]
dict["a"] // 1
```

## Assignment

Assignment uses `=` and is only valid for mutable locations:

- **`var` bindings**: `count = count + 1`
- **Field assignment**: `person.name = "Bob"`
- **Index assignment**: `arr[0] = 99`, `dict["a"] = 2`

Assigning to a `const` binding is a compile error. What a `const` _holds_ stays mutable, though: `const p = Person("Alice", 30)` then `p.name = "Bob"` is allowed, because it is the field being written and not the binding.

The discard pattern `_ = expr` evaluates `expr` and discards the result.

A guarded access is not an assignment target: `user?.name = "Bob"` is a compile error, since `?.` and `!.` may produce no field to assign to.

### Augmented assignment

`+=`, `-=`, `*=`, `/=` and `%=` read the location, apply the operator to what they find and the right operand, and write the result back. They work wherever `=` does — a `var` binding, a field, an index — and mean exactly what the written-out form means, so `+=` concatenates two `String` values as `+` does.

```zirric
var count = 0
count += 1 // 1

const totals = ["a": 1]
totals["a"] *= 3 // ["a": 3]
```

## Closures

Closures are anonymous functions written with `fn(params) { body }`. They share syntax with `fn` declarations but have no name.

```zirric
const add = fn(a, b) {
	return a + b
}
const double = fn(x) {
	return x * 2
}
```

### Capture semantics

Closures capture variables from enclosing scopes. How a variable is captured depends on whether it was declared with `const` or `var`:

- **`const` captures** are captured by value. The closure receives a copy of the value at the time of capture. This is efficient and requires no indirection.

- **`var` captures** are captured by reference through an **upvalue cell**. The compiler wraps the variable in a shared cell so that mutations from either the enclosing scope or the closure are visible to both. Multiple closures capturing the same `var` share the same cell.

```zirric
fn makeCounter() {
	var count = 0
	const increment = fn() {
		count = count + 1
		count
	}

	return increment
}

const counter = makeCounter()
counter() // 1
counter() // 2
```

A loop binding is a `const`, so a closure written inside a loop body captures that iteration's element rather than sharing one binding with the next:

```zirric
const fns = for n <- [1, 2, 3] {
	fn() { n * 10 }
}
fns[0]() // 10
fns[2]() // 30
```

### Return type hints

Closures may declare a return type hint with `->`:

```zirric
const toStr = fn(x: Bool) -> String {
	return if x { "true" } else { "false" }
}
```

## If

`if` comes in two forms: expressions and statements. Both evaluate a condition that must be a `Bool`.

### If expression

An `if` expression produces a value. Each branch contains exactly one expression. An `else` branch is required.

```zirric
const label = if answer == 42 {
	"yes"
} else {
	"no"
}
```

`if` expressions cannot contain `return` statements.

### If statement

An `if` statement executes a branch for its side effects and produces no value. Branches may contain multiple statements, including `return`. The `else` branch is optional.

```zirric
if answer == 42 {
	fmt.fprintln("yes", out)
} else if answer == 0 {
	fmt.fprintln("zero", out)
} else {
	fmt.fprintln("no", out)
}
```

### Else-if chains

Both forms support chaining with `else if`:

```zirric
const tier = if score > 90 {
	"A"
} else if score > 80 {
	"B"
} else {
	"C"
}
```

## For

`for` loops come in three variants and two forms (expression and statement). Statements can be used for side effects, while expressions collect values into an array.

### Collection iteration

Iterates over a collection using `<-`. The binding variable receives each element.

```zirric
for item <- [1, 2, 3] {
	fmt.fprintln(fmt.sprint(item), out)
}
```

See [Type System § Protocol Attributes](/specification/typesystem#protocol-attributes) for values that can be iterated.

### Boolean loop

Evaluates a condition each iteration and continues while it is `true`.

```zirric
for ready {
	doWork()
}
```

### Infinite loop

A `for` with no binding and no condition runs indefinitely until a `break` is reached.

```zirric
for {
	if shouldStop {
		break
	}
}
```

### For expression

A `for` expression collects the values produced by its body into an `Array`. The body is a single expression. `continue` skips a value (does not add to the result). `break` ends collection early. `return` is not allowed.

```zirric
const doubled = for n <- [1, 2, 3] {
	n * 2
}
// doubled is [2, 4, 6]

const odds = for n <- [1, 2, 3, 4] {
	if n % 2 != 0 {
		n
	} else {
		continue
	}
}
// odds is [1, 3]
```

### For statement

A `for` statement executes for side effects and produces no value. The body may contain multiple statements, including `return` if inside a function.

## Switch

`switch` evaluates an expression and matches it against a series of cases. It comes in expression and statement forms.

### Case patterns

Each `case` matches against one of:

- **Value case**: matches when the scrutinee equals the case expression.
- **Type case** (`is Type`): matches when the scrutinee is an instance of the named type. See [Type System § Type Checking](/specification/typesystem#type-checking-with-is).
- **Attribute case** (`is @Attr`): matches when the scrutinee's type carries the named attribute.
- **Wildcard** (`_`): matches any value.

```zirric
switch value {
case 1:
	fmt.fprintln("one", out)
case is String:
	fmt.fprintln("a string", out)
case is @Iterable:
	fmt.fprintln("iterable", out)
case _:
	fmt.fprintln("other", out)
}
```

Cases are evaluated top to bottom. The first matching case executes.

### Switch expression

A `switch` expression produces a value. Each case body is a single expression. A `_` wildcard case is required: a switch expression whose cases all fail has no value to produce, and nothing checks the `is` cases for having covered every member of a union — so write the `_` case even where they appear to.

```zirric
const label = switch code {
case 200:
	"ok"
case 404:
	"not found"
case _:
	"unknown"
}
```

### Switch statement

A `switch` statement executes for side effects. Case bodies may contain multiple statements. The `_` case is optional.

## `is` Expression

The `is` operator tests whether a value belongs to a type or carries an attribute. It produces a `Bool`.

```zirric
value is String // true if value is a String
value is Option // true if value is a member of the Option union
value is @Countable // true if value's type has the @Countable attribute
```

`is` checks work with all [type hint forms](/specification/typesystem#type-hints): named types, union types, and attribute types. See [Type System § Type Checking](/specification/typesystem#type-checking-with-is) for the matching rules.

## `return`, `break`, `continue`

### `return`

Exits the enclosing function immediately, optionally with a value. If no value is given, `void` is returned.

```zirric
fn find(items, target) {
	for item <- items {
		if item == target {
			return item
		}
	}
	return void
}
```

`return` is only valid inside `fn` declarations, closures, and statement-form control flow. It cannot appear inside expression-form `if`, `for`, or `switch`.

### `break`

Exits the enclosing `for` loop immediately. In a `for` expression, the collected array up to that point is the result.

### `continue`

Skips to the next iteration of the enclosing `for` loop. In a `for` expression, no value is added for the current iteration.

## Evaluation Order

- **Arguments** are evaluated left to right before the call.
- **Array elements** are evaluated left to right.
- **Dict entries** are evaluated left to right (key, then value, for each entry).
- **Binary operators** evaluate the left operand first, then the right (except short-circuit `&&` and `||`).
- **Declarations** at the top level are processed in source order. Within a module, all top-level names are visible throughout the file regardless of declaration order (mutual recursion is supported).

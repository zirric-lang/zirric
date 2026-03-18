# Expressions

Zirric is expression-first: most constructs produce a value. This page covers all expression and statement forms, their evaluation semantics, and how control flow works.

For the grammar of each form, see [Syntax § Expressions](/specification/syntax#expressions) and [Syntax § Statements](/specification/syntax#statements).

## Literals

Zirric supports the following literal forms. Each produces a value of the corresponding built-in type.

```zirric
42                 // Int
3.14               // Float
0x8899aa           // Int (hexadecimal)
0o777              // Int (octal)
0b101010           // Int (binary)
1e10               // Float (scientific)
true               // Bool
false              // Bool
"Hello, World!"    // String
[1, 2, 3]          // Array
["key": "value"]   // Dict
```

**Array literals** create a new `Array` value. Elements are evaluated left to right.

**Dict literals** create a new `Dict` value. Keys and values are evaluated left to right. Keys must be `String` values.

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
p.name       // field of a data instance
strings.join // member of a module
```

Accessing a nonexistent field is a runtime error.

### Index access

The `[]` operator accesses elements by index (`Array`, `String`) or by key (`Dict`).

```zirric
const arr = [10, 20, 30]
arr[0]           // 10

const dict = ["a": 1]
dict["a"]        // 1
```

## Assignment

Assignment uses `=` and is only valid for mutable locations:

- **`var` bindings**: `count = count + 1`
- **Field assignment**: `person.name = "Bob"` (on `data` instances with `var` binding)
- **Index assignment**: `arr[0] = 99`

Assigning to a `const` binding is a compile error. The discard pattern `_ = expr` evaluates `expr` and discards the result.

## Closures

Closures are anonymous functions written with `fn(params) { body }`. They share syntax with `fn` declarations but have no name.

```zirric
const add = fn(a, b) { a + b }
const double = fn(x) { x * 2 }
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
    increment
}

const counter = makeCounter()
counter()  // 1
counter()  // 2
```

### Return type hints

Closures may declare a return type hint with `->`:

```zirric
const toStr = fn(x: Int) -> String { "" + x }
```

## If

`if` comes in two forms: expressions and statements. Both evaluate a condition that must be a `Bool`.

### If expression

An `if` expression produces a value. Each branch contains exactly one expression. An `else` branch is required.

```zirric
const label = if answer == 42 { "yes" } else { "no" }
```

`if` expressions cannot contain `return` statements.

### If statement

An `if` statement executes a branch for its side effects and produces no value. Branches may contain multiple statements, including `return`. The `else` branch is optional.

```zirric
if answer == 42 {
    print("yes")
} else if answer == 0 {
    print("zero")
} else {
    print("no")
}
```

### Else-if chains

Both forms support chaining with `else if`:

```zirric
const tier = if score > 90 { "A" } else if score > 80 { "B" } else { "C" }
```

## For

`for` loops come in three variants and two forms (expression and statement).

### Collection iteration

Iterates over a collection using `<-`. The binding variable receives each element.

```zirric
for item <- [1, 2, 3] {
    print(item)
}
```

How iteration works depends on the value's `@Iterable` attribute. See [Type System § Protocol Attributes](/specification/typesystem#protocol-attributes).

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
    if shouldStop { break }
}
```

### For expression

A `for` expression collects the values produced by its body into an `Array`. The body is a single expression. `continue` skips a value (does not add to the result). `break` ends collection early. `return` is not allowed.

```zirric
const doubled = for n <- [1, 2, 3] { n * 2 }
// doubled is [2, 4, 6]

const odds = for n <- [1, 2, 3, 4] {
    if n % 2 != 0 { n } else { continue }
}
// odds is [1, 3]
```

### For statement

A `for` statement executes for side effects and produces no value. The body may contain multiple statements, including `return`.

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
    print("one")
case is String:
    print("a string")
case is @Iterable:
    print("iterable")
case _:
    print("other")
}
```

Cases are evaluated top to bottom. The first matching case executes.

### Switch expression

A `switch` expression produces a value. Each case body is a single expression. A `_` wildcard case is required.

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
value is String        // true if value is a String
value is Option        // true if value is a member of the Option union
value is @Countable    // true if value's type has the @Countable attribute
```

`is` checks work with all [type hint forms](/specification/typesystem#type-hints): named types, union types, and attribute types. See [Type System § Type Checking](/specification/typesystem#type-checking-with-is) for the matching rules.

## `return`, `break`, `continue`

### `return`

Exits the enclosing function immediately, optionally with a value. If no value is given, `void` is returned.

```zirric
fn find(items, target) {
    for item <- items {
        if item == target { return item }
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

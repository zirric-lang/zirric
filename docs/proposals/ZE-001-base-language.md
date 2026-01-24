---
title: ZE-001 - Base Language
description: Defines the base language features of Zirric.
---

# Base Language

::: callout warning <svg xmlns="http://www.w3.org/2000/svg" width="28" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-search-code-icon lucide-search-code"><path d="m13 13.5 2-2.5-2-2.5"/><path d="m21 21-4.3-4.3"/><path d="M9 8.5 7 11l2 2.5"/><circle cx="11" cy="11" r="8"/></svg> In Progress
This proposal is currently being discussed and developed.
Parts might be incomplete or missing in Zirric.
:::

## Introduction

Zirric is an experimental programming language designed to bridge the gap between scripting languages and full-fledged programming languages while keeping its own identity.
Zirric is dynamically, but strongly typed. It aims to be simple yet have batteries included.
Possible long term use cases are all terminal related stuff and text UI applications that might spread to other domains.

In this proposal we define the base language features in detail. All other proposals will build on top of this one.

## Motivation

Zirric is created to address the flaws identified by the [Lithia programming language](https://github.com/vknabel/lithia). In short these were:

- the hardly readable function call syntax
- the bad combination of dynamically, strong typed languages especially with lazy evaluation
- lack of control flow structures
- no obvious way to build modern way of file parsing due to missing type hints or annotations
- performance issues due to the interpreter design, lazy evaluation and the lack of control flow structures
- the `type` expression being limited to types only

Things that proved to be good in Lithia and are kept in Zirric:

- the combination of `union` and `data` types work great together
- `union` was called `enum` in Lithia which was slightly misleading
- the concept of `extern` types
- the concept of `module`, `import`
- the path based module system
- the prelude modules
- the convention of small modules
- the general declaration syntax
- the concept of the `Potfile`
- a single binary that includes all tooling including the lsp
- the tooling experience in regards to the maturity

Additionally there will be a few ideas that worked great in Lithia, but won't in Zirric:

- witnesses had their place, but are now far easier to implement with annotations
- currying is nice and might have its revival in Zirric, but not from the beginning
- immutability conflicts with potential use cases of Zirric

## Non-Goals

Zirric explicitly does not implement generics, interfaces or inheritance.
Zirric does not try to be an embedded or systems language. Competing with other scripting languages in terms of performance is not a goal.
It is not built to mirror existing languages, but tries to find its own way by combining a few concepts to form something larger.

## Proposed Solution

Zirric is an imperative and functional programming language.

Different kinds of declarations are supported:

- `let` variables
- `func` functions
- `union` that group other types
- `data` that define custom data types
- `extern` that define bindings to external libraries
- `annotation` that add metadata to other declarations
- `module` that allows access to the current module
- `import` that allows access to other modules

Besides that there are a few control flow structures:

- `if` expressions and statements
- `switch` expressions and statements
- `for` expressions and statements, that include infinite, boolean and collection based loops
- `break` within loops
- `continue` within loops
- `return` within functions

## Detailed Design

Zirric is a dynamically typed language, but strict when it comes to conversions.
The following sections will all include requirements, examples and EBNF snippets to describe the syntax. The EBNF snippets are not complete and only show the relevant parts.

In general Zirric might introduce type checks at run- or compile time. In these cases these are not considered a breaking change as they replace undefined behavior.

### Expressions

The following precedence groups and operators are supported:

| Precedence    | Operators                        | Associativity |
| ------------- | -------------------------------- | ------------- |
| `LOWEST`      |                                  | None          |
| `LOGICAL_OR`  | `\|\|`                           | Left          |
| `LOGICAL_AND` | `&&`                             | Left          |
| `COMPARISON`  | `==`, `!=`, `<`, `<=`, `>`, `>=` | None          |
| `COALESCING`  | `??` (reserved)                  | Right         |
| `RANGE`       | `..<` (reserved)                 | None          |
| `SUM`         | `+`, `-`                         | Left          |
| `PRODUCT`     | `*`, `/`, `%`                    | Left          |
| `BITWISE`     | `<<` (reserved), `>>` (reserved) | Left          |
| `PREFIX`      | `-x`, `!x`                       | Right         |
| `CALL`        | `fun(x)`                         | Left          |
| `MEMBER`      | `.`, `?.` (reserved)             | Left          |

> **Note:** Operators marked as "(reserved)" are not currently implemented in Zirric. They are reserved for possible future use and may be subject to change or removal in later versions. Their presence in this table does not guarantee future support, but indicates that their syntax is being considered for potential language features.

```zirric
let value = object.field + 2 // member access
fun(x + 1, y * 2) // function call
```

### Literals

The following literals are supported:

```zirric
42                 // Int
3.14               // Float
0x8899aa           // Hex Int
0777               // Octal Int
0b101010           // Binary Int
1e10               // Scientific Float
true               // Bool
false              // Bool
"Hello, World!"    // String
[1, 2, 3]          // Array
{ "key": "value" } // Dict
{ a, b -> a + b }  // Function literal
```

### Variables

Variables can be declared with the `let` keyword.
Variables are only valid within their scope and nested scopes.
Variables don't have types.
Variables can be annotated.
At runtime the values of a variable may be changed.

```zirric
let x = 42
x = 2

_ = x // drops result

@SomeAnnotation()
let y = true
```

```ebnf
decl_let = "_", "=", expression ;
decl_let = [annotation_chain],  "let", identifier, "=", expression ;
```

### Data types

Data types are the most common types in Zirric and store data. In most other languages they are called classes or structs. They are defined by the `data` keyword followed by the type name and a list of fields.

```zirric
data Person {
    name
    age
}
```

To create a new instance of a data type, simply call the type name as a function with the field values as arguments. The order must match the declaration order.

```zirric
let person = Person("John", 42)
_ = person.name // "John"
```

```ebnf
decl_data = "data", type_identifier, [ "{", { decl_field }, "}" ] ;
```

#### Fields

Fields are the building blocks of data types. They are defined by their name and optionally annotations.
To increase the expressiveness, function fields can be defined by adding a function signature after the field name.

In practice this serves just as documentation, as fields can store any value.

```zirric
data Example {
  field1
  @SomeAnnotation()
  field2
  functionField(param1, param2)
}
```

```ebnf
decl_field = [annotation_chain], identifier, [ "(", [ parameter_list ], ")" ] ;
```

### Union types

Union types are used to express that their values can be one of a group of types. In other languages they are also called union types. They are defined by the `union` keyword followed by the type name and a list of types.

As convenience, you can even declare types within the union declaration. These will still be available outside of the union.

```zirric
union JuristicPerson {
    Person
    data Company { // this data will be globally available
        name
        corporateForm
    }
}
```

In this example every `Person` and every `Company` is a `JuristicPerson`.

```ebnf
decl_union = "union", type_identifier, "{", { union_member }, "}" ;
union_member = ( static_reference | decl_data ) ;
```

### Annotation types

Annotations are metadata that can be attached to declarations like `let`, `func`, `data`, `union`, `extern` and `module`.
Instantiations of annotation types can only be created at compile time.
As syntactic sugar non-annotation types can be used as annotations. In this case an annotation of type `Type` will be created with the type as argument. Annotations that are actual annotation types, parenthesis are required.

```zirric
@OtherAnnotation()
annotation SomeAnnotation {
  field1
  @Int field2 // @Type(Int) field2
}
```

```ebnf
decl = [annotation_chain], ( decl_let | decl_func | decl_union | decl_data | decl_annotation | decl_extern_type | decl_extern_func | decl_module ) ;
annotation_chain = annotation, { annotation } ;
annotation = "@", static_reference, [ "(", [ argument_list ], ")" ] ;

decl_annotation = "annotation", type_identifier, [ "{", { decl_field }, "}" ] ;
```

#### Field annotations

Fields of `data` and `annotation` types can be annotated with metadata. These annotations will be processed at compile time and can be accessed at runtime.

```zirric
data Person {
    @String // shorthand for @Type(String)
    @json.HasKey("name")
    name

    @Int // shorthand for @Type(Int)
    @json.HasKey("age")
    age
}
```

#### Parameter annotations

Parameters of functions can be annotated with metadata. These annotations will be processed at compile time and can be accessed at runtime.

```zirric
@Returns(Int)
func add(@Int a, @Int b) {
    return a + b
}
```

#### Accessing annotations

Annotations can be accessed at runtime by using the `reflect` module.

```zirric
import json

let person = Person("John", 42)

let nameAnnotation = reflect.typeOf(person).
    field("name").
    annotation(json.HasKey)
```

> [!attention] Undefined
> The current api definition of the `reflect` module is undefined and will be defined in a separate proposal.

### Extern types

Extern types are built-in types that are implemented in the runtime like `Func`, `String` or `Int`. They are defined by the `extern type` keyword followed by the type name. Optionally you can add a list of fields.

Extern declarations must always be global and cannot be nested.

```zirric
extern type Int
extern type String {
    length
}
```

Each extern type behaves slightly different in terms of how it is created and accessed.
Many types like `String`, `Int`, `Float` and `Dict` will be created by literals, types like `Func` and `Module` by declarations. `Any` on the other hand is more like an `union` containing all types.

```ebnf
decl_extern_type = "extern", "type", type_identifier, [ "{", { decl_field }, "}" ] ;
```

### Extern values

Extern values are values that are implemented in the runtime like `void`. They are defined by the `extern let` keywords followed by the value name.

Extern declarations must always be global and cannot be nested.

```zirric
extern let void
```

### Extern functions

Extern functions are functions that are implemented in the runtime like `print`. They are defined by the `extern func` keywords followed by the function signature.

Extern declarations must always be global and cannot be nested.

```zirric
extern func print(@Has(StringLike) str)
```

Similar to functions these can be called like normal functions.

```ebnf
decl_extern_func = "extern", "func", identifier, "(", [ parameter_list ], ")" ;
```

### Functions

Functions are defined by the `func` keyword followed by the function name, a list of parameters and a body.

```zirric
func add(a, b) {
    return a + b
}
let result = add(1, 2) // result is 3

@Returns(Int)
func add(@Int a, @Int b) {
    return a + b
}

let multiply = { a, b -> 
    a * b
}
let multiline = { a, b -> 
    let result = a * b
    return result + 1
}
```

```ebnf
decl_func = "func", identifier, "(", [ parameter_list ], ")", block ;
parameter_list = parameter, { ",", parameter } ;
parameter = [ annotation_chain ], identifier ;
block = "{", { statement }, "}" ;

func_literal = "{",[ [ parameter_list ], "->" ], block, "}" ;

stmt_return = "return", [ expression ] ;
```

### If expressions and statements

If expressions and statements are used to conditionally execute code. They are defined by the `if` keyword followed by a condition and a body. Optionally you can add `else if` and `else` clauses. Expressions always require an `else` clause.

```zirric
if condition {
    // multiple statements and local declarations are allowed
} else if otherCondition {
    // multiple statements and local declarations are allowed
} else {
    // multiple statements and local declarations are allowed
}

let result = if condition {
    // variables allowed
    let a = 1
    let b = 2
    a + b // but just one expression
} else {
    2
}
```

```ebnf
stmt_if = "if", expression, block, { "else", ( stmt_if | block ) } ;
expr_if = "if", expression, expr_block, { "else", ( expr_if | expr_block ) } ;
```

### Switch expressions and statements

Switch expressions and statements are used to conditionally execute code based on the value of an expression. They are defined by the `switch` keyword followed by an expression and a list of cases. Each case is defined by the `case` keyword followed by a value and a body. Optionally you can add a `_` case. Expressions always require a `_` case.

```zirric
switch value {
case @String: // if value has type String
  //multiple statements and local declarations are allowed
case @Has(Annotation): // if type of value has the annotation
  // multiple statements and local declarations are allowed
case 1:
  // multiple statements and local declarations are allowed
case 2:
  // multiple statements and local declarations are allowed
case _:
  // multiple statements and local declarations are allowed
}

let result = switch value {
case @String:
  // variables allowed, but just one expression
  0
case @Has(Annotation):
  1
case 1:
  2
case _:
  3
}
```

```ebnf
stmt_switch = "switch", expression, "{", { switch_case }, "}" ;
switch_case = "case", ( expression | annotation | "_" ), ":", block ;
```

### For expressions and statements

For expressions and statements are used to iterate over collections, until a condition invalidates or infinitely. They are defined by the `for` keyword followed by a loop definition and a body. The loop definition can be one of the following:

```zirric
for { // infinite loop
    // multiple statements and local declarations are allowed
    if condition {
        break // to exit the loop
    } else if otherCondition {
        continue // to skip to the next iteration
    }
}

for condition { // boolean loop
    // multiple statements and local declarations are allowed
}

for item <- items { // collection loop
    // multiple statements and local declarations are allowed
}

// produces an array
let result = for item <- items { // collection expression
    // variables allowed, but just one expression
    item * 2
}
// produces an array with filtering and breaking
let filtered = for num <- items {
  if num % 13 == 0 {
    break // finish the produced array
  } else if num % 2 == 0 && num % 3 == 0 {
    "fizzbuzz" // appends value
  } else if num % 2 == 0 {
    "fizz"
  } else if num % 3 == 0 {
    "buzz"
  } else {
    continue // skip this value
  }
}
```

```ebnf
stmt_for = "for", [ ( expression | identifier "<-" expression ) ], "{", block, "}" ;
expr_for = "for", [ ( expression | identifier "<-" expression ) ], "{", expr_for_block, "}" ;

stmt_break = "break" ;
stmt_continue = "continue" ;

expr_for_block = { decl }, ( "break" | "continue" | expression ) ;
```

### Modules

Modules are defined by the folder structure on the file system. Each folder is a module. The root module is defined by the folder containing the `Potfile`.
Each module has a corresponding value of type `Module` that can be accessed by the `module` declaration. That way it can also be annotated with metadata.

Declarations that precede with `_` are treated as private and cannot be accessed from other modules. The same applies to nested declarations, imports and module-self references.

```zirric
@Deprecated("Use other module instead")
module examples

func greet() {
    print(examples) // prints the module
}
```

```ebnf
decl_module = "module", identifier ;
```

### Imports

Imports are used to access other modules. They are defined by the `import` keyword followed by the module path separated by dots. Optionally a list of names to import can be specified.

```zirric
import maths
import some.examples {
  func1
}

import alias = some.other.example // import with alias to avoid name clashes
func main() {
    maths.sin(0.5) // requires prefix of module
    func1() // directly accessible
    examples.func2() // others require prefix of module
    alias.func3() // access via alias
}
```

```ebnf
decl_import = "import", [ identifier, "=" ], static_reference, [ "{", import_list, "}" ] ;
import_list = identifier, { ",", identifier } ;
```

## Changes to the Standard Library

This introduces lots of new concepts that will be used by the standard library.

- shims for common extern types like `Int`, `String`, `Char`, `Float`, `Bool`, `Array`, `Dict`, `Func`, `Any`, `AnyType`, `Void` and `Module`
- extern constants like `void`
- annotations for common use cases like `Type`, `Numeric`, `Has`, `Returns` and `Deprecated`, `Countable`, `Iterable`
- data types like `Range`

This also requires the existence of a `reflect` module to be able to access annotations at runtime, but this will be defined in a separate proposal.

### Special Extern Types

- `Any` that can hold any value
- `AnyType` that can hold any type
- `Void` that represents the absence of a value
- `Func` that represents functions
- `Module` that represents modules
- `ModuleType` that represents module types

### Special Annotations

- `@Iterable(iter)` is used by the `for item <- items` syntax to indicate that a type is iterable. Used by compiler and tooling.
- `@Type(type)` that indicates that a value must be of the given type. Used by tooling.
- `@Has(annotation)` that indicates that a type must have the given annotation. Intended for parameters, fields and `case @Has(Annotation)` in switch statements. Not intended for declarations on types. Used by tooling.
- `@Returns(type)` that indicates that a function returns a value of the given type. Used by tooling.
- `@Deprecated(reason)` that indicates that a declaration is deprecated and should not be used anymore. Used by tooling.
- `@Doc(description)` generated by compiler. Contains the documentation comment of a declaration. Used by tooling.

## Acknowledgements

A lot of ideas were taken from existing programming languages like Lithia, Go, Swift, TypeScript, Ruby and Python.

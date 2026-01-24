# Expressions

Zirric is expression-first: most constructs produce a value. This document
captures the base language surface described in `proposals/ZE-001-base-language.md`.

## Literals

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

## Operators and precedence

| Precedence    | Operators                        | Associativity |
| ------------- | -------------------------------- | ------------- |
| `LOWEST`      |                                  | None          |
| `LOGICAL_OR`  | `                                |               |
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

Operators marked as reserved are part of the base language proposal but may not
be implemented yet.

```zirric
let value = object.field + 2
fun(x + 1, y * 2)
```

## Calls and member access

Calls use `name(args...)`. Member access uses `.` and reads fields off `data`
instances or `extern` values.

```zirric
let p = Person("Alice", 30)
print(p.name)
```

## Function literals

Function literals are written as `{ params -> expr }`. The body is a single
expression.

```zirric
let add = { a, b -> a + b }
add(1, 2)
```

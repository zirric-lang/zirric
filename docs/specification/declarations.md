# Declarations

Zirric is driven by declarations. These constructs define values, types, and
modules and are annotated with metadata.

## `let`

`let` introduces a variable. Variables are scoped and may be reassigned.

```zirric
let count = 0
count = count + 1
```

`_ = expr` explicitly discards a value.

## `func`

Functions are declared with `func name(params) { ... }`.

```zirric
func greet(name) {
    "Hello, " + name
}
```

## `data`

`data` declares record-like types with named fields.

```zirric
data Person {
    name
    age
}
```

Instances are created by calling the type name as a function.

```zirric
let alice = Person("Alice", 30)
```

## `union`

`union` defines tagged unions. Types listed inside are available at top level.

```zirric
union Result {
    data Ok { value }
    data Err { error }
}
```

## `extern`

`extern` declares runtime-provided types or functions.

```zirric
extern type String {
    length
}
```

## `annotation`

`annotation` defines metadata schemas used by tooling and the runtime.

```zirric
annotation Returns {
    @Type(AnyType) type
}
```

## `module` and `import`

Modules group declarations. Imports bring other modules into scope.

```zirric
module http

import code.knabel.dev.zirric_lang.zirric.prelude
import code.knabel.dev.zirric_lang.zirric.future.tasks
```

## Annotations on declarations

Annotations attach metadata and behavior to declarations and fields.

```zirric
@Deprecated("use newFn")
func oldFn() { ... }
```

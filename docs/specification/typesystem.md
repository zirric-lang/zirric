# Typesystem

Zirric is dynamically, but strongly typed. In the future it may add type
inference, but today values can flow through any variable, field, or parameter
as long as explicit conversions and runtime checks are satisfied.

The type system stays intentionally small. Zirric supports the following classes
of types:

- `data` types are the most common types. They are used to store data and can be easily created by calling the type name as a function.
- `union` types are used to express that their values can be one of a group of types. In other languages they are also called union types.
- `annotation` types are used to annotate declarations with metadata. They can only be constructed at compile time.
- `extern` types are built-in types that are implemented in the runtime like `Func`, `String` or `Int`.

## Data types

Data types are the most common types in Zirric and store data. In most other languages they are called classes or structs. They are defined by the `data` keyword followed by the type name and a list of fields.

```zirric
data Person {
    name
    age
}
```

To create a new instance of a data type, simply call the type name as a function with the field values as arguments.

```zirric
let person = Person("John", 42)
```

### Fields

Fields are the building blocks of data types. They are defined by their name and optionally annotations.
To increase the expressiveness, function fields can be defined by adding a function signature after the field name.

```zirric
data Greetable {
    greeting(ofValue) // function field
}
```

### Field annotations

Fields can be annotated with metadata. These annotations will be processed at compile time and can be accessed at runtime.

```zirric
data Person {
    @String
    @json.HasKey("name")
    name

    @Int
    @json.HasKey("age")
    age
}
```

## Union types

Union types are used to express that their values can be one of a group of types. In other languages they are also called union types. They are defined by the `union` keyword followed by the type name and a list of types.
As convenience, you can even declare types within the union declaration. These will still be available outside of the union.

```zirric
union JuristicPerson {
    Person
    data Company {
        name
        corporateForm
    }
}
```

In this example every `Person` and every `Company` is a `JuristicPerson`.

To discriminate union values, use `switch` with annotation cases:

```zirric
func nameOf(juristic) {
    switch juristic {
    case @Person:
        juristic.name
    case @Company:
        juristic.name + " " + juristic.corporateForm
    case _:
        "unknown"
    }
}
```

## Annotation types

Annotation types are used to annotate declarations with metadata. They can only be constructed at compile time.
They are defined by the `annotation` keyword followed by the type name and a list of fields.

```zirric
module json

annotation HasKey {
    name
}
```

To annotate a declaration, start with the `@` symbol followed by the annotation type name and a list of field values.

```zirric
import json

data Person {
    @Type(String)
    @json.HasKey("name")
    name

    @Int // shorthand for @Type(Int)
    @json.HasKey("age")
    age
}
```

### Accessing annotations

Annotations can be accessed at runtime by using the `reflect` module.

```zirric
import json
import code.knabel.dev.zirric_lang.zirric.future.reflect

let person = Person("John", 42)
let personType = reflect.typeOf(person)
let fields = reflect.fieldsOf(personType)
let nameAnnotation = reflect.annotation(fields[0], json.HasKey)
```

## Protocol-like annotations

The prelude defines annotations such as `@Countable` and `@Iterable` to describe
capabilities of types. The compiler and tooling use these annotations to drive
loop behavior and helper utilities.

```zirric
@Countable({ v -> v.length })
@Iterable(_arrayIterate)
extern type Array {
    @Int length
}
```

## Common prelude patterns

The prelude includes small data structures such as `Result` and `Optional` that
encode success/failure and nullable values without relying on implicit nulls.

## Extern types

Extern types are built-in types that are implemented in the runtime like `Func`, `String` or `Int`. They are defined by the `extern` keyword followed by the type name. Optionally you can add a list of fields.

```zirric
extern Int

extern String {
    length
}
```

Each extern type behaves slightly different in terms of how it is created and accessed.
Many types like `String`, `Int`, `Float` and `Dict` will be created by literals, types like `Func` and `Module` by declarations. `Any` on the other hand is more like an `union` containing all types.

> _**Note:**_ The `extern` keyword is also used to declare functions provided by the compiler like `extern print(str)`.

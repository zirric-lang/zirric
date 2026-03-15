# Attributes

Zirric supports annotating declarations with metadata via attributes. Attributes are processed
at compile time and can be accessed at runtime via reflection.

## Syntax

Attributes are written as `@Name(args...)`. Arguments are comma-separated
expressions, including literals, identifiers, or function literals.

```zirric
@AttrName("argument", 123)
data MyData
```

Before being able to use an attribute, it must be declared and imported:

```zirric
mod my

attr AttrName {
    field
    field2
}

// other module
import my

@my.AttrName
data SomeDeclaration
```

### Syntactic sugar

If the attribute has no arguments, the parentheses can be omitted.

```zirric
@AttrName
data MyData
```

If the referenced type is not an attribute, it will implicitly be converted to
an `@Type` attribute.

```zirric
data MyData {
    @String // equivalent to @Type(String)
    field
}
```

## Built-in attributes

The prelude defines common attributes used by the compiler, runtime, and tools:

- `@Type(T)`: declares a value or field to be of type `T`.
- `@Has(AttributeType)`: requires values to carry a specific attribute.
- `@Returns(T)`: declares a function return type.
- `@Default(value)`: documents a default value.
- `@Doc(text)`: attaches a documentation string.
- `@Deprecated(reason)`: marks a declaration as deprecated.
- `@Numeric`: marks values that can be converted to numbers.

### `@Type`

`@Type` specifies the type of a declaration. It is also used implicitly when
annotating with non-attribute types like `@String`.

```zirric
data MyData {
    @Type(String)
    field
}

// or

data MyData {
    @Type("String")
    field
}
```

# Annotations

Zirric supports annotating declarations with metadata. Annotations are processed
at compile time and can be accessed at runtime via reflection.

## Syntax

Annotations are written as `@Name(args...)`. Arguments are comma-separated
expressions, including literals, identifiers, or function literals.

```zirric
@AnnotationName("argument", 123)
data MyData
```

Before being able to use an annotation, it must be declared and imported:

```zirric
module my

annotation AnnotationName {
    field
    field2
}

// other module
import my

@my.AnnotationName
data SomeDeclaration
```

### Syntactic sugar

If the annotation has no arguments, the parentheses can be omitted.

```zirric
@AnnotationName
data MyData
```

If the referenced type is not an annotation, it will implicitly be converted to
an `@Type` annotation.

```zirric
data MyData {
    @String // equivalent to @Type(String)
    field
}
```

## Built-in annotations

The prelude defines common annotations used by the compiler, runtime, and tools:

- `@Type(T)`: declares a value or field to be of type `T`.
- `@Has(AnnotationType)`: requires values to carry a specific annotation.
- `@Returns(T)`: declares a function return type.
- `@Default(value)`: documents a default value.
- `@Doc(text)`: attaches a documentation string.
- `@Deprecated(reason)`: marks a declaration as deprecated.
- `@Numeric`: marks values that can be converted to numbers.

### `@Type`

`@Type` specifies the type of a declaration. It is also used implicitly when
annotating with non-annotation types like `@String`.

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

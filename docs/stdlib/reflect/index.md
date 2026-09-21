---
title: Reflect
description: Inspecting modules, types, fields and values at runtime.
---

# Module `reflect`

> What a program can learn about itself while running.

```zirric
import reflect
```

|            |                                                                                                                                                                                                                  |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `reflect`                                                                                                                                                                                                        |
| **Source** | [`reflect/reflect.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/reflect/reflect.zirr), [`reflect/types.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/reflect/types.zirr) |

`reflect` reads the structure a program declared: the members of a module, the fields of a `data` type, the members of a `union`, and the type of a value.

It is what makes attribute-driven libraries possible. [`coding`](../coding/index.md) decides how to encode a type by walking its [`Field`](#field) values and reading the attributes written on each one — and attributes are read off a `Field` exactly as off any other value, so `coding.Name(field).text` and `field is @coding.Ignore` both work.

[`TypeRef`](#typeref) describes a type _hint_ as it was written, which is not the same as what a value turns out to be. Hints are not enforced at runtime, so a `TypeRef` tells you what a declaration promises.

## Contents

- **Unions** — [`TypeRef`](#typeref)
- **Data** — [`Field`](#field)
- **Functions** — [`moduleName`](#modulename), [`members`](#members), [`memberNames`](#membernames), [`member`](#member), [`hasMember`](#hasmember), [`typeName`](#typename), [`isDataType`](#isdatatype), [`isUnionType`](#isuniontype), [`isAttributeType`](#isattributetype), [`isInstance`](#isinstance), [`construct`](#construct), [`fieldValues`](#fieldvalues), [`fieldsOf`](#fieldsof), [`unionMembers`](#unionmembers), [`typeOf`](#typeof)

---

## Unions

### `TypeRef` {#typeref}

<small>`reflect/types.zirr:14`</small>

```zirric
union TypeRef {
	// A type referred to by name, e.g. String or Person.
	data NamedType {
		// The name as written, which is qualified when the declaration qualified it, e.g. "String" or "prelude.Option".
		name: String
		// The type the name refers to, or None when it refers to nothing that exists as a value at runtime.
		resolved: Option
	}

	// An array type, e.g. [String].
	data ArrayType {
		element: TypeRef
	}

	// A dict type, e.g. [String: Int].
	data DictType {
		key: TypeRef
		value: TypeRef
	}

	// A function type, e.g. fn(String) -> Int.
	data FuncType {
		// One TypeRef per parameter, UnknownType where a parameter had no hint.
		parameters: [TypeRef]
		// The declared result, which is an UnknownType when the function declares none.
		returns: TypeRef
	}

	// An attribute constraint, e.g. @Iterable @Countable.
	data AttrsType {
		// One NamedType per required attribute.
		attributes: [TypeRef]
	}

	// No type hint was written.
	data UnknownType
}
```

A type hint as it was written. Hints are not enforced at runtime, so a TypeRef describes what a declaration promises rather than what a value turns out to be.

#### Cases

| Case          | Interpretation                                      |
| ------------- | --------------------------------------------------- |
| `NamedType`   | A type referred to by name, e.g. String or Person.  |
| `ArrayType`   | An array type, e.g. [String].                       |
| `DictType`    | A dict type, e.g. [String: Int].                    |
| `FuncType`    | A function type, e.g. fn(String) -> Int.            |
| `AttrsType`   | An attribute constraint, e.g. @Iterable @Countable. |
| `UnknownType` | No type hint was written.                           |

---

## Data

### `Field` {#field}

<small>`reflect/types.zirr:5`</small>

```zirric
data Field {
	// The name the field was declared under.
	name: String
	// The type hint written on the field, which is an UnknownType when none was.
	declaredType: TypeRef
}
```

A field of a data type, carrying the attributes written on it. Attributes are read off a Field exactly as off any other value, so `coding.Name(field).text` and `field is @coding.Ignore` both work.

#### Fields

| Field          | Signature               | Description                                                                |
| -------------- | ----------------------- | -------------------------------------------------------------------------- |
| `name`         | `name: String`          | The name the field was declared under.                                     |
| `declaredType` | `declaredType: TypeRef` | The type hint written on the field, which is an UnknownType when none was. |

---

## Functions

### `moduleName` {#modulename}

<small>`reflect/reflect.zirr:4`</small>

```zirric
extern fn moduleName(m: Module) -> String
```

The canonical name of a module, e.g. "code.knabel.dev.zirric_lang.zirric.arrays".

---

### `members` {#members}

<small>`reflect/reflect.zirr:7`</small>

```zirric
extern fn members(m: Module) -> [Any]
```

Every public member of m, ordered by name.

---

### `memberNames` {#membernames}

<small>`reflect/reflect.zirr:10`</small>

```zirric
extern fn memberNames(m: Module) -> [String]
```

The name of every public member of m, in the same order as members.

---

### `member` {#member}

<small>`reflect/reflect.zirr:15`</small>

```zirric
fn member(m: Module, name: String) -> Option
```

The public member of m called name, or None if it has none.

---

### `hasMember` {#hasmember}

<small>`reflect/reflect.zirr:26`</small>

```zirric
fn hasMember(m: Module, name: String) -> Bool
```

Whether m has a public member called name.

---

### `typeName` {#typename}

<small>`reflect/reflect.zirr:31`</small>

```zirric
extern fn typeName(t: Any) -> String
```

The declared name of a type, or the empty String for anything that is not a type.

---

### `isDataType` {#isdatatype}

<small>`reflect/reflect.zirr:34`</small>

```zirric
extern fn isDataType(t: Any) -> Bool
```

Whether t is a data type, and so has fields to enumerate.

---

### `isUnionType` {#isuniontype}

<small>`reflect/reflect.zirr:37`</small>

```zirric
extern fn isUnionType(t: Any) -> Bool
```

Whether t is a union type, and so has members to enumerate.

---

### `isAttributeType` {#isattributetype}

<small>`reflect/reflect.zirr:40`</small>

```zirric
extern fn isAttributeType(t: Any) -> Bool
```

Whether t is an attribute type, and so can be called on a value to read that attribute.

---

### `isInstance` {#isinstance}

<small>`reflect/reflect.zirr:44`</small>

```zirric
extern fn isInstance(value: Any, t: Any) -> Bool
```

Whether a value is of a type, decided exactly as the `is` operator decides it. Unlike `is`, the type is a value here, so it can come from reflection rather than from source.

---

### `construct` {#construct}

<small>`reflect/reflect.zirr:47`</small>

```zirric
extern fn construct(t: Any, values: [Any]) -> Result
```

Builds a value of a data type from its fields, in declaration order, or `Err` if the count does not match.

---

### `fieldValues` {#fieldvalues}

<small>`reflect/reflect.zirr:50`</small>

```zirric
extern fn fieldValues(value: Any) -> [Any]
```

The field values of a data value, in declaration order and so aligned with `fieldsOf`, or an empty array for anything else.

---

### `fieldsOf` {#fieldsof}

<small>`reflect/reflect.zirr:53`</small>

```zirric
extern fn fieldsOf(t: Any) -> [Field]
```

The fields of a data type, in declaration order, or an empty array for anything else.

---

### `unionMembers` {#unionmembers}

<small>`reflect/reflect.zirr:56`</small>

```zirric
extern fn unionMembers(t: Any) -> [Any]
```

The member types of a union type, in declaration order, or an empty array for anything else.

---

### `typeOf` {#typeof}

<small>`reflect/reflect.zirr:61`</small>

```zirric
fn typeOf(value: Any) -> Option
```

The type of a value, or None for a value whose type has no runtime representation.

---

## See also

- [`reflect.packages`](./packages/index.md) — discovering the modules of the running package.
- [`coding`](../coding/index.md) — the main consumer of all this.
- [Type System](/specification/typesystem) — what the hints mean.

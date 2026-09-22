---
title: Reflect
description: Inspecting modules, types, fields and values at runtime.
---

# Module `reflect`

```zirric
import reflect
```

|            |                                                                                                                                                                                                                                                                                                                                     |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `reflect`                                                                                                                                                                                                                                                                                                                           |
| **Source** | [`reflect/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/reflect/module-docs.zirr), [`reflect/reflect.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/reflect/reflect.zirr), [`reflect/types.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/reflect/types.zirr) |

> What a program can learn about itself while running.

`reflect` reads the structure a program declared: the members of a module, the fields of a `data` type, the members of a `union`, and the type of a value.

It is what makes attribute-driven libraries possible. [`coding`](../coding/index.md) decides how to encode a type by walking its [`Field`](#field) values and reading the attributes written on each one — and attributes are read off a [`Field`](#field) exactly as off any other value, so `coding.Name(field).text` and `field is @coding.Ignore` both work.

[`TypeRef`](#typeref) describes a type _hint_ as it was written, which is not the same as what a value turns out to be. Hints are not enforced at runtime, so a [`TypeRef`](#typeref) tells you what a declaration promises.

## Contents

- **Unions** — [`DeclarationKind`](#declarationkind), [`TypeRef`](#typeref)
- **Data** — [`ArrayType`](#arraytype), [`AttrDecl`](#attrdecl), [`AttrsType`](#attrstype), [`ConstDecl`](#constdecl), [`DataDecl`](#datadecl), [`Declaration`](#declaration-data), [`DictType`](#dicttype), [`Field`](#field), [`FuncDecl`](#funcdecl), [`FuncType`](#functype), [`Module`](#module), [`NamedType`](#namedtype), [`Source`](#source), [`TypeDecl`](#typedecl), [`UnionDecl`](#uniondecl), [`UnknownDecl`](#unknowndecl), [`UnknownType`](#unknowntype), [`VarDecl`](#vardecl)
- **Functions** — [`construct`](#construct), [`declaration`](#declaration-fn), [`docs`](#docs), [`fieldValues`](#fieldvalues), [`fieldsOf`](#fieldsof), [`hasMember`](#hasmember), [`isAttributeType`](#isattributetype), [`isDataType`](#isdatatype), [`isInstance`](#isinstance), [`isUnionType`](#isuniontype), [`member`](#member), [`memberNames`](#membernames), [`members`](#members), [`moduleName`](#modulename), [`moduleOf`](#moduleof), [`typeName`](#typename), [`typeOf`](#typeof), [`unionMembers`](#unionmembers)

---

## Unions

### `DeclarationKind` {#declarationkind}

<small>`reflect/types.zirr:42`</small>

```zirric
union DeclarationKind {
	DataDecl
	UnionDecl
	AttrDecl
	TypeDecl
	FuncDecl
	ConstDecl
	VarDecl
	UnknownDecl
}
```

The form a declaration was written in, which is the keyword that introduced it.

#### Cases

| Case          | Interpretation                                        |
| ------------- | ----------------------------------------------------- |
| `DataDecl`    | A data declaration.                                   |
| `UnionDecl`   | A union declaration.                                  |
| `AttrDecl`    | An attribute declaration.                             |
| `TypeDecl`    | An extern type declaration.                           |
| `FuncDecl`    | A function, declared with fn or extern fn.            |
| `ConstDecl`   | A constant, declared with const or extern const.      |
| `VarDecl`     | A variable, declared with var.                        |
| `UnknownDecl` | A declaration in a form reflection does not describe. |

---

### `TypeRef` {#typeref}

<small>`reflect/types.zirr:81`</small>

```zirric
union TypeRef {
	NamedType
	ArrayType
	DictType
	FuncType
	AttrsType
	UnknownType
}
```

A type hint as it was written.
Hints are not enforced at runtime, so a TypeRef describes what a declaration promises rather than what a value turns out to be.

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

### `ArrayType` {#arraytype}

<small>`reflect/types.zirr:91`</small>

```zirric
data ArrayType {
	element: TypeRef
}
```

An array type, e.g. [String].

#### Fields

| Field     | Description |
| --------- | ----------- |
| `element` |             |

---

### `AttrDecl` {#attrdecl}

<small>`reflect/types.zirr:50`</small>

```zirric
data AttrDecl
```

An attribute declaration.

---

### `AttrsType` {#attrstype}

<small>`reflect/types.zirr:110`</small>

```zirric
data AttrsType {
	attributes: [TypeRef]
}
```

An attribute constraint, e.g. @Iterable @Countable.

#### Fields

| Field        | Description                           |
| ------------ | ------------------------------------- |
| `attributes` | One NamedType per required attribute. |

---

### `ConstDecl` {#constdecl}

<small>`reflect/types.zirr:59`</small>

```zirric
data ConstDecl
```

A constant, declared with const or extern const.

---

### `DataDecl` {#datadecl}

<small>`reflect/types.zirr:44`</small>

```zirric
data DataDecl
```

A data declaration.

---

### `Declaration` {#declaration-data}

<small>`reflect/types.zirr:28`</small>

```zirric
data Declaration {
	name: String
	docs: String
	kind: DeclarationKind
	signature: String
	source: Source
}
```

A public declaration of a module, carrying the attributes written on it.
Attributes are read off a Declaration exactly as off any other value, so `tests.Comment(declaration).text` and `declaration is @Deprecated` both work, even for a declaration whose value can carry none, such as a const holding an Int.

#### Fields

| Field       | Description                                                                                       |
| ----------- | ------------------------------------------------------------------------------------------------- |
| `name`      | The name it was declared under, which is the name [`member`](#member) takes.                      |
| `docs`      | The comment written above it, or the empty String when none was.                                  |
| `kind`      | The form it was written in.                                                                       |
| `signature` | The declaration as it was written, without its body, e.g. "extern fn typeName(t: Any) -> String". |
| `source`    | Where it was written.                                                                             |

---

### `DictType` {#dicttype}

<small>`reflect/types.zirr:96`</small>

```zirric
data DictType {
	key: TypeRef
	value: TypeRef
}
```

A dict type, e.g. [String: Int].

#### Fields

| Field   | Description |
| ------- | ----------- |
| `key`   |             |
| `value` |             |

---

### `Field` {#field}

<small>`reflect/types.zirr:70`</small>

```zirric
data Field {
	name: String
	docs: String
	declaredType: TypeRef
}
```

A field of a data type, carrying the attributes written on it.
Attributes are read off a Field exactly as off any other value, so `coding.Name(field).text` and `field is @coding.Ignore` both work.

#### Fields

| Field          | Description                                                                |
| -------------- | -------------------------------------------------------------------------- |
| `name`         | The name the field was declared under.                                     |
| `docs`         | The comment written above the field, or the empty String when none was.    |
| `declaredType` | The type hint written on the field, which is an UnknownType when none was. |

---

### `FuncDecl` {#funcdecl}

<small>`reflect/types.zirr:56`</small>

```zirric
data FuncDecl
```

A function, declared with fn or extern fn.

---

### `FuncType` {#functype}

<small>`reflect/types.zirr:102`</small>

```zirric
data FuncType {
	parameters: [TypeRef]
	returns: TypeRef
}
```

A function type, e.g. fn(String) -> Int.

#### Fields

| Field        | Description                                                                   |
| ------------ | ----------------------------------------------------------------------------- |
| `parameters` | One TypeRef per parameter, UnknownType where a parameter had no hint.         |
| `returns`    | The declared result, which is an UnknownType when the function declares none. |

---

### `Module` {#module}

<small>`reflect/types.zirr:5`</small>

```zirric
data Module {
	name: String
	docs: String
	imports: [String]
	sources: [String]
	declarations: [Declaration]
}
```

What a module is, as metadata: everything reflection can say about it without reaching into the values it exports.
`AnyModule` is the module itself, which members are looked up on; this is what it declared.

#### Fields

| Field          | Description                                                                                                            |
| -------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `name`         | The canonical name, e.g. "reflect.packages".                                                                           |
| `docs`         | The comment written above each of its `mod` declarations, joined in file name order.                                   |
| `imports`      | The name of every module it imports, ordered by name.                                                                  |
| `sources`      | The path of every file it was read from, ordered by name.                                                              |
| `declarations` | Every public declaration, ordered by name and so aligned with [`members`](#members) and [`memberNames`](#membernames). |

---

### `NamedType` {#namedtype}

<small>`reflect/types.zirr:83`</small>

```zirric
data NamedType {
	name: String
	resolved: Option
}
```

A type referred to by name, e.g. String or Person.

#### Fields

| Field      | Description                                                                                                   |
| ---------- | ------------------------------------------------------------------------------------------------------------- |
| `name`     | The name as written, which is qualified when the declaration qualified it, e.g. "String" or "prelude.Option". |
| `resolved` | The type the name refers to, or None when it refers to nothing that exists as a value at runtime.             |

---

### `Source` {#source}

<small>`reflect/types.zirr:19`</small>

```zirric
data Source {
	path: String
	line: Int
}
```

Where a declaration was written.

#### Fields

| Field  | Description                                                                                       |
| ------ | ------------------------------------------------------------------------------------------------- |
| `path` | The file it was read from, named as the module reached it, e.g. "reflect/types.zirr".             |
| `line` | The line the declaration starts on, counted from 1, or 0 when it was not read from a file at all. |

---

### `TypeDecl` {#typedecl}

<small>`reflect/types.zirr:53`</small>

```zirric
data TypeDecl
```

An extern type declaration.

---

### `UnionDecl` {#uniondecl}

<small>`reflect/types.zirr:47`</small>

```zirric
data UnionDecl
```

A union declaration.

---

### `UnknownDecl` {#unknowndecl}

<small>`reflect/types.zirr:65`</small>

```zirric
data UnknownDecl
```

A declaration in a form reflection does not describe.

---

### `UnknownType` {#unknowntype}

<small>`reflect/types.zirr:116`</small>

```zirric
data UnknownType
```

No type hint was written.

---

### `VarDecl` {#vardecl}

<small>`reflect/types.zirr:62`</small>

```zirric
data VarDecl
```

A variable, declared with var.

---

## Functions

### `construct` {#construct}

<small>`reflect/reflect.zirr:79`</small>

```zirric
extern fn construct(t: Any, values: [Any]) -> Result
```

Builds a value of a data type from its fields, in declaration order, or `Err` if the count does not match.

---

### `declaration` {#declaration-fn}

<small>`reflect/reflect.zirr:18`</small>

```zirric
fn declaration(m: Module, name: String) -> Option
```

The declaration of m called name, or None if m has no public member of that name.
m is the metadata rather than the module itself, so that looking several up costs one pass over the declarations.

---

### `docs` {#docs}

<small>`reflect/reflect.zirr:49`</small>

```zirric
fn docs(value: Any) -> String
```

The comment written above the declaration a value came from, or the empty String when none was written.
A module, a type, an attribute and a function each answer for their own declaration; a Module, a Declaration or a Field answers for the declaration it describes, which is the only way to reach the documentation of a const, whose value is an ordinary value with no declaration behind it.

---

### `fieldValues` {#fieldvalues}

<small>`reflect/reflect.zirr:82`</small>

```zirric
extern fn fieldValues(value: Any) -> [Any]
```

The field values of a data value, in declaration order and so aligned with [`fieldsOf`](#fieldsof), or an empty array for anything else.

---

### `fieldsOf` {#fieldsof}

<small>`reflect/reflect.zirr:85`</small>

```zirric
extern fn fieldsOf(t: Any) -> [Field]
```

The fields of a data or attribute type, in declaration order, or an empty array for anything else.

---

### `hasMember` {#hasmember}

<small>`reflect/reflect.zirr:41`</small>

```zirric
fn hasMember(m: AnyModule, name: String) -> Bool
```

Whether m has a public member called name.

---

### `isAttributeType` {#isattributetype}

<small>`reflect/reflect.zirr:72`</small>

```zirric
extern fn isAttributeType(t: Any) -> Bool
```

Whether t is an attribute type, and so can be called on a value to read that attribute, as well as having fields to enumerate.

---

### `isDataType` {#isdatatype}

<small>`reflect/reflect.zirr:66`</small>

```zirric
extern fn isDataType(t: Any) -> Bool
```

Whether t is a data type, and so has fields to enumerate.

---

### `isInstance` {#isinstance}

<small>`reflect/reflect.zirr:76`</small>

```zirric
extern fn isInstance(value: Any, t: Any) -> Bool
```

Whether a value is of a type, decided exactly as the `is` operator decides it.
Unlike `is`, the type is a value here, so it can come from reflection rather than from source.

---

### `isUnionType` {#isuniontype}

<small>`reflect/reflect.zirr:69`</small>

```zirric
extern fn isUnionType(t: Any) -> Bool
```

Whether t is a union type, and so has members to enumerate.

---

### `member` {#member}

<small>`reflect/reflect.zirr:30`</small>

```zirric
fn member(m: AnyModule, name: String) -> Option
```

The public member of m called name, or None if it has none.

---

### `memberNames` {#membernames}

<small>`reflect/reflect.zirr:10`</small>

```zirric
extern fn memberNames(m: AnyModule) -> [String]
```

The name of every public member of m, in the same order as members.

---

### `members` {#members}

<small>`reflect/reflect.zirr:7`</small>

```zirric
extern fn members(m: AnyModule) -> [Any]
```

Every public member of m, ordered by name.

---

### `moduleName` {#modulename}

<small>`reflect/reflect.zirr:4`</small>

```zirric
extern fn moduleName(m: AnyModule) -> String
```

The canonical name of a module, e.g. "code.knabel.dev.zirric_lang.zirric.arrays".

---

### `moduleOf` {#moduleof}

<small>`reflect/reflect.zirr:14`</small>

```zirric
extern fn moduleOf(m: AnyModule) -> Module
```

What m declared, gathered in one value: its name, its documentation, what it imports, the files it was read from, and every public declaration.
A Declaration answers what an exported value cannot: which keyword declared it, where it was written, and the attributes written on it even where the value itself can carry none.

---

### `typeName` {#typename}

<small>`reflect/reflect.zirr:63`</small>

```zirric
extern fn typeName(t: Any) -> String
```

The declared name of a type, or the empty String for anything that is not a type.

---

### `typeOf` {#typeof}

<small>`reflect/reflect.zirr:93`</small>

```zirric
fn typeOf(value: Any) -> Option
```

The type of a value, or None for a value whose type has no runtime representation.

---

### `unionMembers` {#unionmembers}

<small>`reflect/reflect.zirr:88`</small>

```zirric
extern fn unionMembers(t: Any) -> [Any]
```

The member types of a union type, in declaration order, or an empty array for anything else.

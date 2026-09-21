---
title: Strings
description: Searching, slicing, splitting and transforming text.
---

# Module `strings`

> Text operations that count characters, not bytes.

```zirric
import strings
```

|            |                                                                                                           |
| ---------- | --------------------------------------------------------------------------------------------------------- |
| **Module** | `strings`                                                                                                 |
| **Source** | [`strings/strings.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/strings/strings.zirr) |

`strings` works on [`Like`](#like) — a [`String`](../prelude/index.md#string) or a single [`Char`](../prelude/index.md#char) — so a one-character value can be passed anywhere text is expected.

Positions here are character positions. [`count`](#count) returns the number of UTF-8 code points, which is not the same as `len()`: `len("café")` is 5 bytes, `count("café")` is 4 characters. Indexing a `String` directly gives bytes; [`charAt`](#charat) gives characters.

## Contents

- **Unions** — [`Like`](#like)
- **Functions** — [`from`](#from), [`count`](#count), [`charAt`](#charat), [`slice`](#slice), [`range`](#range), [`isEmpty`](#isempty), [`isDigit`](#isdigit), [`isLetter`](#isletter), [`isSpace`](#isspace), [`contains`](#contains), [`hasPrefix`](#hasprefix), [`hasSuffix`](#hassuffix), [`firstIndexOf`](#firstindexof), [`lastIndexOf`](#lastindexof), [`replace`](#replace), [`replaceFirst`](#replacefirst), [`replaceLast`](#replacelast), [`split`](#split), [`join`](#join), [`concat`](#concat), [`repeat`](#repeat), [`toUpper`](#toupper), [`toLower`](#tolower), [`trim`](#trim), [`trimPrefix`](#trimprefix), [`trimSuffix`](#trimsuffix), [`quote`](#quote), [`unquote`](#unquote)

---

## Unions

### `Like` {#like}

<small>`strings/strings.zirr:6`</small>

```zirric
union Like {
	Char
	String
}
```

A single character or a sequence of characters, so a `Char` can be passed anywhere a `String` is expected.

#### Cases

| Case     | Interpretation      |
| -------- | ------------------- |
| `Char`   | One character.      |
| `String` | Any number of them. |

---

## Functions

### `from` {#from}

<small>`strings/strings.zirr:12`</small>

```zirric
extern fn from(v: Like) -> String
```

Normalizes a Char or String to String.

---

### `count` {#count}

<small>`strings/strings.zirr:16`</small>

```zirric
extern fn count(v: Like) -> Int
```

Returns the number of characters (runes) in v. Unlike len(), which counts bytes, this counts UTF-8 code points — len("café") is 5, count("café") is 4.

---

### `charAt` {#charat}

<small>`strings/strings.zirr:19`</small>

```zirric
extern fn charAt(v: Like, index: Int) -> Char
```

Returns the character at index (0-based, counting characters, not bytes).

---

### `slice` {#slice}

<small>`strings/strings.zirr:23`</small>

```zirric
extern fn slice(v: Like, start: Int, end: Int) -> String
```

Returns the characters of v from start (inclusive) to end (exclusive), counting characters, not bytes.

---

### `range` {#range}

<small>`strings/strings.zirr:26`</small>

```zirric
fn range(v: Like, r: ranges.Like) -> String
```

Returns the characters of v selected by r.

---

### `isEmpty` {#isempty}

<small>`strings/strings.zirr:38`</small>

```zirric
fn isEmpty(v: Like) -> Bool
```

Returns whether v has no characters.

---

### `isDigit` {#isdigit}

<small>`strings/strings.zirr:44`</small>

```zirric
extern fn isDigit(v: Like) -> Bool
```

Returns whether v is not empty and every character is a decimal digit. Unicode-aware — isDigit('٣') is true, isDigit("") is false.

---

### `isLetter` {#isletter}

<small>`strings/strings.zirr:48`</small>

```zirric
extern fn isLetter(v: Like) -> Bool
```

Returns whether v is not empty and every character is a letter. Unicode-aware — isLetter('é') is true, isLetter("") is false.

---

### `isSpace` {#isspace}

<small>`strings/strings.zirr:52`</small>

```zirric
extern fn isSpace(v: Like) -> Bool
```

Returns whether v is not empty and every character is whitespace. Unicode-aware — a non-breaking space counts, isSpace("") is false.

---

### `contains` {#contains}

<small>`strings/strings.zirr:55`</small>

```zirric
extern fn contains(v: Like, needle: Like) -> Bool
```

Returns whether needle occurs anywhere in v.

---

### `hasPrefix` {#hasprefix}

<small>`strings/strings.zirr:58`</small>

```zirric
extern fn hasPrefix(v: Like, prefix: Like) -> Bool
```

Returns whether v starts with prefix.

---

### `hasSuffix` {#hassuffix}

<small>`strings/strings.zirr:61`</small>

```zirric
extern fn hasSuffix(v: Like, suffix: Like) -> Bool
```

Returns whether v ends with suffix.

---

### `firstIndexOf` {#firstindexof}

<small>`strings/strings.zirr:67`</small>

```zirric
fn firstIndexOf(v: Like, needle: Like) -> Option
```

Returns the character index of needle's first occurrence in v, or None if absent.

---

### `lastIndexOf` {#lastindexof}

<small>`strings/strings.zirr:77`</small>

```zirric
fn lastIndexOf(v: Like, needle: Like) -> Option
```

Returns the character index of needle's last occurrence in v, or None if absent.

---

### `replace` {#replace}

<small>`strings/strings.zirr:87`</small>

```zirric
extern fn replace(v: Like, target: Like, replacement: Like) -> String
```

Replaces every occurrence of target in v with replacement.

---

### `replaceFirst` {#replacefirst}

<small>`strings/strings.zirr:90`</small>

```zirric
extern fn replaceFirst(v: Like, target: Like, replacement: Like) -> String
```

Replaces only the first occurrence of target in v with replacement.

---

### `replaceLast` {#replacelast}

<small>`strings/strings.zirr:93`</small>

```zirric
extern fn replaceLast(v: Like, target: Like, replacement: Like) -> String
```

Replaces only the last occurrence of target in v with replacement.

---

### `split` {#split}

<small>`strings/strings.zirr:96`</small>

```zirric
extern fn split(v: Like, separator: Like) -> [String]
```

Splits v on every occurrence of separator.

---

### `join` {#join}

<small>`strings/strings.zirr:99`</small>

```zirric
extern fn join(parts: [Like], separator: Like) -> String
```

Joins parts into a single String, with separator between each.

---

### `concat` {#concat}

<small>`strings/strings.zirr:102`</small>

```zirric
extern fn concat(parts: [Like]) -> String
```

Concatenates every part into a single String, in order.

---

### `repeat` {#repeat}

<small>`strings/strings.zirr:105`</small>

```zirric
extern fn repeat(v: Like, n: Int) -> String
```

Returns v repeated n times.

---

### `toUpper` {#toupper}

<small>`strings/strings.zirr:109`</small>

```zirric
extern fn toUpper(v: Like) -> Like
```

Returns v with every letter converted to uppercase. Preserves whether v was a Char or a String — toUpper('a') is 'A', toUpper("a") is "A".

---

### `toLower` {#tolower}

<small>`strings/strings.zirr:113`</small>

```zirric
extern fn toLower(v: Like) -> Like
```

Returns v with every letter converted to lowercase. Preserves whether v was a Char or a String — toLower('A') is 'a', toLower("A") is "a".

---

### `trim` {#trim}

<small>`strings/strings.zirr:116`</small>

```zirric
extern fn trim(v: Like) -> String
```

Returns v with leading and trailing whitespace removed.

---

### `trimPrefix` {#trimprefix}

<small>`strings/strings.zirr:119`</small>

```zirric
extern fn trimPrefix(v: Like, prefix: Like) -> String
```

Returns v with a leading prefix removed, if present.

---

### `trimSuffix` {#trimsuffix}

<small>`strings/strings.zirr:122`</small>

```zirric
extern fn trimSuffix(v: Like, suffix: Like) -> String
```

Returns v with a trailing suffix removed, if present.

---

### `quote` {#quote}

<small>`strings/strings.zirr:127`</small>

```zirric
extern fn quote(v: Like) -> String
```

Returns v as a double-quoted string literal, escaping the characters that need it, so the result reads back as Zirric source. A Char is quoted as a String — quote('a') is the same as quote("a").

---

### `unquote` {#unquote}

<small>`strings/strings.zirr:132`</small>

```zirric
extern fn unquote(v: Like) -> Result
```

Reads a quoted literal, returning Ok with the text it denotes and Err when v is not one. Both literal forms are accepted, "\"hi\"" and "'a'", and quote is inverted exactly.

---

## See also

- [`bytes`](../bytes/index.md) — the byte-level equivalents.
- [`fmt`](../fmt/index.md) — turning arbitrary values into strings.
- [`ranges`](../ranges/index.md) — the range values [`range`](#range) accepts.

---
title: Strings
description: Searching, slicing, splitting and transforming text.
---

# Module `strings`

```zirric
import strings
```

|            |                                                                                                                                                                                                                              |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Module** | `strings`                                                                                                                                                                                                                    |
| **Source** | [`strings/module-docs.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/strings/module-docs.zirr), [`strings/strings.zirr`](https://code.knabel.dev/zirric-lang/zirric/src/branch/main/strings/strings.zirr) |

> Text operations that count characters, not bytes.

`strings` works on [`Like`](#like) — a [`prelude.String`](../prelude/index.md#string) or a single [`prelude.Char`](../prelude/index.md#char) — so a one-character value can be passed anywhere text is expected.

Positions here are character positions. [`count`](#count) returns the number of UTF-8 code points, which is not the same as `len()`: `len("café")` is 5 bytes, `count("café")` is 4 characters. Indexing a `String` directly gives bytes; [`charAt`](#charat) gives characters.

## Dependencies

- [`ranges`](../ranges/index.md)

---

## Contents

- **Unions** — [`Like`](#like)
- **Functions** — [`charAt`](#charat), [`concat`](#concat), [`contains`](#contains), [`count`](#count), [`firstIndexOf`](#firstindexof), [`from`](#from), [`hasPrefix`](#hasprefix), [`hasSuffix`](#hassuffix), [`isDigit`](#isdigit), [`isEmpty`](#isempty), [`isLetter`](#isletter), [`isSpace`](#isspace), [`join`](#join), [`lastIndexOf`](#lastindexof), [`quote`](#quote), [`range`](#range), [`repeat`](#repeat), [`replace`](#replace), [`replaceFirst`](#replacefirst), [`replaceLast`](#replacelast), [`slice`](#slice), [`split`](#split), [`toLower`](#tolower), [`toUpper`](#toupper), [`trim`](#trim), [`trimPrefix`](#trimprefix), [`trimSuffix`](#trimsuffix), [`unquote`](#unquote)

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

A single character or a sequence of characters.

#### Cases

| Case     | Interpretation                                                        |
| -------- | --------------------------------------------------------------------- |
| `Char`   | A single character from a string.                                     |
| `String` | A regular String. Can be indexed to get the Byte. Iterates over Char. |

---

## Functions

### `charAt` {#charat}

<small>`strings/strings.zirr:18`</small>

```zirric
extern fn charAt(v: Like, index: Int) -> Char
```

Returns the character at index (0-based, counting characters, not bytes).

---

### `concat` {#concat}

<small>`strings/strings.zirr:100`</small>

```zirric
extern fn concat(parts: [Like]) -> String
```

Concatenates every part into a single String, in order.

---

### `contains` {#contains}

<small>`strings/strings.zirr:53`</small>

```zirric
extern fn contains(v: Like, needle: Like) -> Bool
```

Returns whether needle occurs anywhere in v.

---

### `count` {#count}

<small>`strings/strings.zirr:15`</small>

```zirric
extern fn count(v: Like) -> Int
```

Returns the number of characters (runes) in v. Unlike len(), which counts bytes, this counts UTF-8 code points — len("café") is 5, count("café") is 4.

---

### `firstIndexOf` {#firstindexof}

<small>`strings/strings.zirr:65`</small>

```zirric
fn firstIndexOf(v: Like, needle: Like) -> Option
```

Returns the character index of needle's first occurrence in v, or None if absent.

---

### `from` {#from}

<small>`strings/strings.zirr:12`</small>

```zirric
extern fn from(v: Like) -> String
```

Normalizes a Char or String to String.

---

### `hasPrefix` {#hasprefix}

<small>`strings/strings.zirr:56`</small>

```zirric
extern fn hasPrefix(v: Like, prefix: Like) -> Bool
```

Returns whether v starts with prefix.

---

### `hasSuffix` {#hassuffix}

<small>`strings/strings.zirr:59`</small>

```zirric
extern fn hasSuffix(v: Like, suffix: Like) -> Bool
```

Returns whether v ends with suffix.

---

### `isDigit` {#isdigit}

<small>`strings/strings.zirr:42`</small>

```zirric
extern fn isDigit(v: Like) -> Bool
```

Returns whether v is not empty and every character is a decimal digit.
Unicode-aware — isDigit('٣') is true, isDigit("") is false.

---

### `isEmpty` {#isempty}

<small>`strings/strings.zirr:36`</small>

```zirric
fn isEmpty(v: Like) -> Bool
```

Returns whether v has no characters.

---

### `isLetter` {#isletter}

<small>`strings/strings.zirr:46`</small>

```zirric
extern fn isLetter(v: Like) -> Bool
```

Returns whether v is not empty and every character is a letter.
Unicode-aware — isLetter('é') is true, isLetter("") is false.

---

### `isSpace` {#isspace}

<small>`strings/strings.zirr:50`</small>

```zirric
extern fn isSpace(v: Like) -> Bool
```

Returns whether v is not empty and every character is whitespace.
Unicode-aware — a non-breaking space counts, isSpace("") is false.

---

### `join` {#join}

<small>`strings/strings.zirr:97`</small>

```zirric
extern fn join(parts: [Like], separator: Like) -> String
```

Joins parts into a single String, with separator between each.

---

### `lastIndexOf` {#lastindexof}

<small>`strings/strings.zirr:75`</small>

```zirric
fn lastIndexOf(v: Like, needle: Like) -> Option
```

Returns the character index of needle's last occurrence in v, or None if absent.

---

### `quote` {#quote}

<small>`strings/strings.zirr:121`</small>

```zirric
extern fn quote(v: Like) -> String
```

Returns v as a double-quoted string literal, escaping the characters that need it, so the result reads back as Zirric source. A Char is quoted as a String — quote('a') is the same as quote("a").

---

### `range` {#range}

<small>`strings/strings.zirr:24`</small>

```zirric
fn range(v: Like, r: ranges.Like) -> String
```

Returns the characters of v selected by r.

---

### `repeat` {#repeat}

<small>`strings/strings.zirr:103`</small>

```zirric
extern fn repeat(v: Like, n: Int) -> String
```

Returns v repeated n times.

---

### `replace` {#replace}

<small>`strings/strings.zirr:85`</small>

```zirric
extern fn replace(v: Like, target: Like, replacement: Like) -> String
```

Replaces every occurrence of target in v with replacement.

---

### `replaceFirst` {#replacefirst}

<small>`strings/strings.zirr:88`</small>

```zirric
extern fn replaceFirst(v: Like, target: Like, replacement: Like) -> String
```

Replaces only the first occurrence of target in v with replacement.

---

### `replaceLast` {#replacelast}

<small>`strings/strings.zirr:91`</small>

```zirric
extern fn replaceLast(v: Like, target: Like, replacement: Like) -> String
```

Replaces only the last occurrence of target in v with replacement.

---

### `slice` {#slice}

<small>`strings/strings.zirr:21`</small>

```zirric
extern fn slice(v: Like, start: Int, end: Int) -> String
```

Returns the characters of v from start (inclusive) to end (exclusive), counting characters, not bytes.

---

### `split` {#split}

<small>`strings/strings.zirr:94`</small>

```zirric
extern fn split(v: Like, separator: Like) -> [String]
```

Splits v on every occurrence of separator.

---

### `toLower` {#tolower}

<small>`strings/strings.zirr:109`</small>

```zirric
extern fn toLower(v: Like) -> Like
```

Returns v with every letter converted to lowercase. Preserves whether v was a Char or a String — toLower('A') is 'a', toLower("A") is "a".

---

### `toUpper` {#toupper}

<small>`strings/strings.zirr:106`</small>

```zirric
extern fn toUpper(v: Like) -> Like
```

Returns v with every letter converted to uppercase. Preserves whether v was a Char or a String — toUpper('a') is 'A', toUpper("a") is "A".

---

### `trim` {#trim}

<small>`strings/strings.zirr:112`</small>

```zirric
extern fn trim(v: Like) -> String
```

Returns v with leading and trailing whitespace removed.

---

### `trimPrefix` {#trimprefix}

<small>`strings/strings.zirr:115`</small>

```zirric
extern fn trimPrefix(v: Like, prefix: Like) -> String
```

Returns v with a leading prefix removed, if present.

---

### `trimSuffix` {#trimsuffix}

<small>`strings/strings.zirr:118`</small>

```zirric
extern fn trimSuffix(v: Like, suffix: Like) -> String
```

Returns v with a trailing suffix removed, if present.

---

### `unquote` {#unquote}

<small>`strings/strings.zirr:124`</small>

```zirric
extern fn unquote(v: Like) -> Result
```

Reads a quoted literal, returning Ok with the text it denotes and Err when v is not one. Both literal forms are accepted, "\"hi\"" and "'a'", and quote is inverted exactly.

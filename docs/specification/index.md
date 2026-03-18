# Specification

This section is the authoritative reference for the Zirric programming language. It covers every construct the language supports — syntax, semantics, and runtime behavior.

## How to use this section

The specification is split into four pages. Start with whichever matches your question:

- **[Syntax](/specification/syntax)** — The formal grammar. Use this when you need to know _what is valid to write_: tokens, keywords, operator precedence, and the EBNF productions for every construct. No semantic explanations — just structure.

- **[Declarations](/specification/declarations)** — How named entities are introduced: `const`, `var`, `fn`, `data`, `union`, `attr`, `extern`, `mod`, `import`. Covers scoping rules, attribute application and validation, and where type hints appear on declarations.

- **[Expressions](/specification/expressions)** — How values are computed: literals, operators, calls, member and index access, closures with capture semantics, and all control flow (`if`, `for`, `switch`, `is`, `return`, `break`, `continue`). Covers expression vs. statement forms and evaluation order.

- **[Type System](/specification/typesystem)** — How types work at runtime: the four type categories (`data`, `union`, `extern type`, `attr`), construction and identity, union membership, type hints and composite type expressions, `is` matching rules, and protocol attributes like `@Countable` and `@Iterable`.

## Quick reference

| I want to know…                                | Go to                                                                                                                                                         |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Is this syntax valid?                          | [Syntax](/specification/syntax)                                                                                                                               |
| What operators exist and their precedence?     | [Syntax § Operator Precedence](/specification/syntax#operator-precedence)                                                                                     |
| How does `const` vs `var` capture in closures? | [Declarations § const](/specification/declarations#const) / [Expressions § Closures](/specification/expressions#closures)                                     |
| How do I declare and apply attributes?         | [Declarations § attr](/specification/declarations#attr) / [Declarations § Attributes on Declarations](/specification/declarations#attributes-on-declarations) |
| How does `switch case is` matching work?       | [Expressions § Switch](/specification/expressions#switch)                                                                                                     |
| What does `is @Attr` check?                    | [Type System § Attribute Types](/specification/typesystem#attribute-types)                                                                                    |
| What type hint forms are available?            | [Type System § Type Hints](/specification/typesystem#type-hints)                                                                                              |
| How do unions and membership work?             | [Type System § Union Types](/specification/typesystem#union-types)                                                                                            |
| How does `for` iteration work under the hood?  | [Expressions § For](/specification/expressions#for) / [Type System § Protocol Attributes](/specification/typesystem#protocol-attributes)                      |

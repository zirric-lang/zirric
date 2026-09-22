---
title: Zirric Evolution Proposals
description: A collection of Zirric Evolution Proposals (ZEPs) documenting changes and additions to the Zirric language and ecosystem.
---

# Zirric Evolution Proposals (ZE)

This directory contains Zirric Evolution Proposals. Each proposal documents a change or addition to the Zirric language or ecosystem. Take care to notice the status of each proposal.

- **Draft**: The proposal is in an early stage and may undergo significant changes.
- **In Progress**: The proposal is being actively worked on and refined.
- **Implemented**: The proposal has been implemented in the Zirric language or ecosystem.
- **Rejected**: The proposal has been reviewed and decided against implementation.

## Proposals

| Proposal                                                       | Title                               | Status      | Info                        |
| -------------------------------------------------------------- | ----------------------------------- | ----------- | --------------------------- |
| [ZE-001](/proposals/ZE-001-base-language)                      | Base Language                       | Implemented | [v0.1.0](/changelog/v0.1.0) |
| [ZE-002](/proposals/ZE-002-the-cavefile)                       | The Cavefile                        | Implemented | [v0.1.0](/changelog/v0.1.0) |
| [ZE-003](/proposals/ZE-003-named-data-construction)            | Named Data Construction             | Rejected    |                             |
| [ZE-004](/proposals/ZE-004-Variadic-Attributes)                | Variadic Attributes                 | Rejected    |                             |
| [ZE-005](/proposals/ZE-005-Mixin-Type-Declarations)            | Mixin Type Declarations             | Draft       | Outdated                    |
| [ZE-006](/proposals/ZE-006-Attribute-Based-Parsing-System)     | Attribute-Based Parsing System      | Rejected    | Superseded by ZE-021        |
| [ZE-007](/proposals/ZE-007-attribute-binding)                  | Attribute Binding                   | Draft       |                             |
| [ZE-008](/proposals/ZE-008-error-handling)                     | Error Handling                      | Implemented | [v0.1.0](/changelog/v0.1.0) |
| [ZE-009](/proposals/ZE-009-option-values)                      | Option Values                       | Implemented | [v0.1.0](/changelog/v0.1.0) |
| [ZE-010](/proposals/ZE-010-iterable)                           | Iterable                            | Implemented | [v0.1.0](/changelog/v0.1.0) |
| [ZE-011](/proposals/ZE-011-zirric-cli)                         | Zirric CLI                          | Implemented | [v0.1.0](/changelog/v0.1.0) |
| [ZE-012](/proposals/ZE-012-type-and-returns-sugar)             | Type and Returns Sugar              | Rejected    |                             |
| [ZE-013](/proposals/ZE-013-mutability-and-constants)           | Mutability and Constants            | Implemented | [v0.1.0](/changelog/v0.1.0) |
| [ZE-014](/proposals/ZE-014-attribute-and-declaration-keywords) | Attribute and Declaration Keywords  | Implemented | [v0.1.0](/changelog/v0.1.0) |
| [ZE-015](/proposals/ZE-015-extern-type-constructors)           | Extern Type Constructors            | Draft       |                             |
| [ZE-016](/proposals/ZE-016-closure-syntax)                     | Unified Function and Closure Syntax | Implemented | [v0.1.0](/changelog/v0.1.0) |
| [ZE-017](/proposals/ZE-017-type-hints)                         | Type Hints and Type Matching        | Implemented | [v0.1.0](/changelog/v0.1.0) |
| [ZE-018](/proposals/ZE-018-io-fmt-os)                          | I/O, Formatting, and OS             | Implemented | [v0.1.0](/changelog/v0.1.0) |
| [ZE-019](/proposals/ZE-019-result-and-option-sugar)            | Result and Option Sugar             | Implemented | [v0.1.0](/changelog/v0.1.0) |
| [ZE-020](/proposals/ZE-020-standard-library)                   | Standard Library                    | Implemented | [v0.1.0](/changelog/v0.1.0) |
| [ZE-021](/proposals/ZE-021-encoding-and-decoding)              | Encoding and Decoding               | Implemented | [v0.1.0](/changelog/v0.1.0) |
| [ZE-022](/proposals/ZE-022-static-checks)                      | Static Checks                       | Implemented | [v0.1.0](/changelog/v0.1.0) |
| [ZE-023](/proposals/ZE-023-code-formatting)                    | Code Formatting                     | Implemented | [v0.1.0](/changelog/v0.1.0) |
| [ZE-024](/proposals/ZE-024-qualified-module-names)             | Qualified Module Names              | Implemented | [v0.1.0](/changelog/v0.1.0) |

## Submitting Proposals

To create a new proposal:

1. Copy [ZE-000-template.md](/proposals/ZE-000-template).
2. Rename the file to follow the scheme `ZE-000-name-of-the-proposal.md`.
3. Link the proposal in this index file.
4. Add an entry to the `tasks/docmd/docmd.config.js`.
5. Fill out the template and open a pull request.

## Caveats

Please note that proposals are subject to change and may not be implemented as initially described.

The presented EBNF grammar snippets are only used for illustration purposes and may not reflect the final syntax of the language. They are intended to convey the general structure and ideas of the proposals, rather than serving as exact specifications. In the end, the parser implementation is the single source of truth for the language syntax, while the [tree-sitter grammar](https://code.knabel.dev/zirric-lang/tree-sitter-zirric) serves as a more compact reference for the structure of the language, while not being an exact specification.

When Zirric undergoes significant changes, existing proposals will most likely not be updated. Only in rare cases, existing proposals might be updated to avoid confusion. This effort will only be made for important or related proposals.

In the end, proposals are a witness of their time and should be treated as such. They provide insight into the evolution of the language and the rationale behind certain design decisions, but they are not necessarily indicative of the current state or future direction of the language.

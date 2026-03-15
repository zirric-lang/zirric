---
title: Zirric Evolution Proposals
description: A collection of Zirric Evolution Proposals (ZEPs) documenting changes and additions to the Zirric language and ecosystem.
---

# Zirric Evolution Proposals (ZE)

This directory contains Zirric Evolution Proposals. Each proposal documents a change
or addition to the Zirric language or ecosystem. Take care to notice the status of each proposal.

- <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-icon lucide-circle"><circle cx="12" cy="12" r="10"/></svg> **Draft**: The proposal is in an early stage and may undergo significant changes.
- <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-dot-icon lucide-circle-dot"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="3" fill="currentColor" stroke="none"/></svg> **In Progress**: The proposal is being actively worked on and refined.
- <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-check-icon lucide-circle-check"><circle cx="12" cy="12" r="10"/><path d="m9 12 2 2 4-4"/></svg> **Implemented**: The proposal has been implemented in the Zirric language or ecosystem.
- <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-circle-x-icon lucide-circle-x"><circle cx="12" cy="12" r="10"/><path d="m15 9-6 6"/><path d="m9 9 6 6"/></svg> **Rejected**: The proposal has been reviewed and decided against implementation.

## Proposals

| Proposal                                                       | Title                              | Status      | Author                                      |
| -------------------------------------------------------------- | ---------------------------------- | ----------- | ------------------------------------------- |
| [ZE-001](/proposals/ZE-001-base-language)                      | Base Language                      | In Progress | [@vknabel](https://github.com/vknabel)      |
| [ZE-002](/proposals/ZE-002-the-cavefile)                       | The Cavefile                       | In Progress | [@vknabel](https://github.com/vknabel)      |
| [ZE-003](/proposals/ZE-003-named-data-construction)            | Named Data Construction            | Rejected    | [@vknabel](https://code.knabel.dev/vknabel) |
| [ZE-004](/proposals/ZE-004-Variadic-Arguments)                 | Variadic Arguments                 | Draft       | [@vknabel](https://code.knabel.dev/vknabel) |
| [ZE-005](/proposals/ZE-005-Mixin-Type-Declarations)            | Mixin Type Declarations            | Draft       | [@vknabel](https://code.knabel.dev/vknabel) |
| [ZE-006](/proposals/ZE-006-Annotation-Based-Parsing-System)    | Attribute-Based Parsing System     | Draft       | [@vknabel](https://code.knabel.dev/vknabel) |
| [ZE-007](/proposals/ZE-007-attribute-binding)                 | Attribute Binding                  | Draft       | [@vknabel](https://code.knabel.dev/vknabel) |
| [ZE-008](/proposals/ZE-008-error-handling)                     | Error Handling                     | In Progress | [@vknabel](https://code.knabel.dev/vknabel) |
| [ZE-009](/proposals/ZE-009-option-values)                      | Option Values                      | In Progress | [@vknabel](https://code.knabel.dev/vknabel) |
| [ZE-010](/proposals/ZE-010-iterable)                           | Iterable                           | In Progress | [@vknabel](https://code.knabel.dev/vknabel) |
| [ZE-011](/proposals/ZE-011-zirric-cli)                         | Zirric CLI                         | In Progress | [@vknabel](https://code.knabel.dev/vknabel) |
| [ZE-012](/proposals/ZE-012-type-and-returns-sugar)             | Type and Returns Sugar             | In Progress | [@vknabel](https://code.knabel.dev/vknabel) |
| [ZE-013](/proposals/ZE-013-mutability-and-constants)           | Mutability and Constants           | Draft       | [@vknabel](https://code.knabel.dev/vknabel) |
| [ZE-014](/proposals/ZE-014-attribute-and-declaration-keywords) | Attribute and Declaration Keywords | Implemented | [@vknabel](https://code.knabel.dev/vknabel) |
| [ZE-015](/proposals/ZE-015-extern-type-constructors)           | Extern Type Constructors           | Draft       | [@vknabel](https://code.knabel.dev/vknabel) |

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

When Zirric undergoes significant changes, existing proposals will most likely not be updated.
Only in rare cases, existing proposals might be updated to avoid confusion. This effort will only be made for important or related proposals.

In the end, proposals are a witness of their time and should be treated as such. They provide insight into the evolution of the language and the rationale behind certain design decisions, but they are not necessarily indicative of the current state or future direction of the language.

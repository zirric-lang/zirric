---
title: Zirric Evolution Proposals
description: A collection of Zirric Evolution Proposals (ZEPs) documenting changes and additions to the Zirric language and ecosystem.
---

# Zirric Evolution Proposals (ZE)

This directory contains Zirric Evolution Proposals. Each proposal documents a change
or addition to the Zirric language or ecosystem. Take care to notice the status of each proposal.

- <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-search-slash-icon lucide-search-slash"><path d="m13.5 8.5-5 5"/><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg> **Draft**: The proposal is in an early stage and may undergo significant changes.
- <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-search-code-icon lucide-search-code"><path d="m13 13.5 2-2.5-2-2.5"/><path d="m21 21-4.3-4.3"/><path d="M9 8.5 7 11l2 2.5"/><circle cx="11" cy="11" r="8"/></svg> **In Progress**: The proposal is being actively worked on and refined.
- <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-search-check-icon lucide-search-check"><path d="m8 11 2 2 4-4"/><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg> **Implemented**: The proposal has been implemented in the Zirric language or ecosystem.
- <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-search-x-icon lucide-search-x"><path d="m13.5 8.5-5 5"/><path d="m8.5 8.5 5 5"/><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg> **Rejected**: The proposal has been reviewed and decided against implementation.

## Proposals

| Proposal                                                    | Title                           | Status      | Author                                                                             |
| ----------------------------------------------------------- | ------------------------------- | ----------- | ---------------------------------------------------------------------------------- |
| [ZE-001](/proposals/ZE-001-base-language)                   | Base Language                   | In Progress | [@vknabel](https://github.com/vknabel)                                             |
| [ZE-002](/proposals/ZE-002-the-cavefile)                    | The Cavefile                    | In Progress | [@vknabel](https://github.com/vknabel), [@blushling](https://github.com/blushling) |
| [ZE-003](/proposals/ZE-003-named-data-construction)         | Named Data Construction         | Rejected    | [@vknabel](https://code.knabel.dev/vknabel)                                        |
| [ZE-004](/proposals/ZE-004-Variadic-Arguments)              | Variadic Arguments              | Draft       | [@vknabel](https://code.knabel.dev/vknabel)                                        |
| [ZE-005](/proposals/ZE-005-Mixin-Type-Declarations)         | Mixin Type Declarations         | Draft       | [@vknabel](https://code.knabel.dev/vknabel)                                        |
| [ZE-006](/proposals/ZE-006-Annotation-Based-Parsing-System) | Annotation-Based Parsing System | Draft       | [@vknabel](https://code.knabel.dev/vknabel)                                        |

## Submitting Proposals

To create a new proposal:

1. Copy [ZE-000-template.md](/proposals/ZE-000-template).
2. Rename the file to follow the scheme `ZE-000-name-of-the-proposal.md`.
3. Link the proposal in this index file.
4. Add an entry to the `docmd.config.js`.
5. Fill out the template and open a pull request.

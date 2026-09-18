# AGENTS.md

## Project Overview

Zirric is an experimental programming language implemented in Go with a bytecode compiler and virtual machine. The repository contains:

- **Language Type**: Programming language implementation (lexer, parser, compiler, VM)
- **Primary Language**: Go 1.24+
- **Size**: Medium-sized project (~18 test files, 17 Go packages)
- **Target**: Educational/experimental programming language with modern features

### Core Components

- **Lexer & Parser**: Tokenize and parse Zirric source code into AST
- **Analyzer**: Resolve symbols, validate static references, and assign IDs
- **Compiler**: Compile AST to bytecode for virtual machine
- **Virtual Machine**: Execute bytecode with stack-based architecture
- **Runtime**: Built-in types (Array, Bool, Int, String) and standard library
- **Package Management**: Cavefile-based dependency management with registries

## Understanding the language

### Where to look first (authoritative Zirric source)

- **Language proposals**: `docs/proposals/ZE-001-base-language.md` (core syntax/semantics) and `docs/proposals/ZE-002-the-cavefile.md` (package manifest + tasks). Recent implemented proposals include `docs/proposals/ZE-002-the-cavefile.md` (Cavefile dependencies and tasks), `docs/proposals/ZE-013-mutability-and-constants.md` (const/var), `docs/proposals/ZE-016-closure-syntax.md` (fn closures), and `docs/proposals/ZE-017-type-hints.md` (type hints and is-matching). ZE-011 (`docs/proposals/ZE-011-zirric-cli.md`, the CLI) is still In Progress. For future-facing features, see `docs/proposals/ZE-004-Variadic-Arguments.md` and `docs/proposals/ZE-005-Mixin-Type-Declarations.md`.
- **Standard library Zirric sources**: `prelude/shim.zirr` (core types and values), `prelude/attributes.zirr` (attribute system), `prelude/countable.zirr` (protocol-like attributes), `prelude/result.zirr` (Result/Optional patterns), `future/reflect/stub.zirr` (reflection surface).
- **Cavefile schema and tasks**: `cave/manifest.zirr` and `tasks/manifest.zirr` define the attribute-driven dependency/task model used by the package manager.
- **Example manifest**: `examples/project/Cavefile` shows real-world dependency + task declarations.

### Core mental model (intuition)

- **Declarations**: Zirric is declaration-driven (`var`, `const`, `fn`, `data`, `union`, `extern`, `attr`, `mod`, `import`), with attributes as the primary metadata mechanism.
- **Dynamic but strict**: Values are dynamic, yet conversions are explicit; there are optional type hints like `const x: Int` and `fn(p: @HasAttr) -> String`.
- **Data and unions**: `data` defines record-like types with named fields; `union` are a declared nominal supertype consisting of a fixed set of existing types; values are implicitly usable as a union if their concrete type is a member (often with nested `data` members).
- **Attributes are first-class**: Behaviors (defaults, docs, protocols) are expressed via attributes in `prelude/attributes.zirr`. Type information uses type hints (`: T`, `-> T`) rather than attributes.
- **Collection protocols**: `@Countable`/`@Iterable` in `prelude/countable.zirr` describe the “protocols” used by loops and helpers.
- **Cavefile is just Zirric**: Dependency and task manifests are Zirric `data` declarations annotated with `@cave.Dependencies` and `@tasks.*` (see `cave/manifest.zirr` and `tasks/manifest.zirr`).

## Build & Validation Instructions

### Prerequisites

- Go 1.24.0 or newer (go.mod targets 1.24; CI uses 1.25)
- All dependencies are managed via go.mod - no additional tools required

### Essential Commands

**ALWAYS run these commands from the repository root.**

#### Format (Required)

```bash
task fmt
```

- **Purpose**: Run Go + dprint formatting fixes
- **Always run**: After making changes, before final validation

#### Check (Required)

```bash
task check
```

- **Purpose**: Run formatting checks, lint, vet, build, and tests
- **Expected**: Clean exit with no output besides command logs

#### Build (Required before testing)

```bash
go build -v ./...
```

- **Time**: ~30-60 seconds on first run, ~5-10 seconds on subsequent runs
- **Purpose**: Compile all packages to verify syntax and dependencies
- **Always succeed**: Should exit with code 0; `-v` prints packages as they build

#### Test (Critical validation)

```bash
go test ./...
```

- **Time**: ~2-5 seconds
- **Purpose**: Run all unit tests across 18 test files
- **Expected**: All tests pass, ~10-15 packages tested
- **Note**: Some packages show "[no test files]" - this is normal

#### Test with Race Detection (Recommended for concurrency changes)

```bash
go test -race ./...
```

- **Time**: ~10-15 seconds
- **Purpose**: Detect race conditions in concurrent code
- **Use when**: Making changes to VM, runtime, or concurrent operations

#### Format Check (Always required)

```bash
task check
```

- **Includes**: `gofmt -l .` and `dprint check --config dprint.json`
- **Critical**: Must pass before committing - GitHub Actions will fail otherwise

#### Format Fix (When formatting fails)

```bash
task fmt
```

- **Purpose**: Fix Go and other formatter-supported files

#### Vet (Recommended)

```bash
go vet ./...
```

- **Time**: ~2-3 seconds
- **Purpose**: Check for suspicious constructs
- **Should pass**: Clean exit with no output

#### Module Maintenance (If dependency issues)

```bash
go mod tidy
```

- **Use when**: Adding/removing dependencies or getting module errors
- **Note**: May download additional test dependencies

### Command Sequences That Work

#### Clean Development Build

```bash
go clean -cache && go build -v ./... && go test ./...
```

#### Full Validation Sequence

```bash
task check
```

#### Fix Formatting + Test

```bash
task fmt && go test ./...
```

### Continuous Integration

GitHub Actions workflow (`.github/workflows/go.yml`):

1. **Go Setup**: Uses Go 1.25
2. **Build**: `go build -v ./...`
3. **Test**: `go test -v ./...`

**Critical**: Both build and test must pass for PR approval. The workflow runs on every push and PR to main branch.

## Project Layout & Architecture

### Repository Structure

```
/
├── .github/workflows/go.yml    # CI/CD pipeline
├── README.md                   # Project documentation
├── LICENSE                     # Mozilla Public License 2.0
├── go.mod                      # Go module definition
├── grammar.ebnf                # Formal grammar specification
├── docs/                       # Language documentation
│   ├── tooling/compiler.md     # Bytecode & VM architecture
│   ├── specification/          # Language specification docs
│   ├── proposals/              # Language evolution proposals (ZE-*)
│   ├── changelog/              # Release history (index.md + per-release files)
│   └── stdlib/                 # Standard library docs
├── examples/project/           # Example Zirric project
├── prelude/                     # Core standard library sources
├── future/                      # Future standard library modules
│   ├── prelude/                 # Proposed prelude extensions (.zirr)
│   ├── reflect/                 # Reflection surface (.zirr)
│   ├── cave/                    # Cavefile manifest schema (.zirr)
│   └── tasks/                   # Cavefile task attributes (.zirr)
├── cmd/                        # CLI entrypoints
├── pkg/                        # Core Go packages
│   ├── ast/                    # Abstract Syntax Tree definitions
│   ├── analyzer/               # Analyzer passes (symbol resolution, IDs)
│   ├── lexer/                  # Tokenization
│   ├── parser/                 # Parse tokens to AST
│   ├── compiler/               # Compile AST to bytecode
│   ├── op/                     # Bytecode operation definitions
│   ├── vm/                     # Virtual machine execution
│   ├── runtime/                # Built-in types and runtime system
│   ├── cavefile/               # Package management structures
│   ├── registry/               # Module/package registries
│   ├── syncheck/               # Syntax validation
│   ├── token/                  # Token definitions
│   ├── version/                # Semantic versioning
│   ├── world/                  # OS interaction abstractions
│   └── pkgmanager/             # Package management logic
```

### Key Files for Code Changes

- **`pkg/ast/`**: Add new AST nodes when extending language syntax
- **`pkg/lexer/lexer.go`**: Modify for new token types
- **`pkg/parser/parser.go`**: Update for new language constructs
- **`pkg/compiler/compiler.go`**: Add compilation logic for new features
- **`pkg/vm/vm.go`**: Extend VM for new bytecode operations
- **`pkg/runtime/prelude-*.go`**: Built-in type implementations
- **`prelude/*.zirr`**: Core types and attributes
- **`cave/manifest.zirr`**: Cavefile dependency schema
- **`tasks/manifest.zirr`**: Cavefile task schema
- **`pkg/op/defs.go`**: Define new bytecode operations

### Testing Patterns

- **Unit tests**: `*_test.go` files alongside source
- **Test data**: Inline test cases in table-driven tests
- **Compiler tests**: `compiler/compiler_test.go` - bytecode validation
- **Parser tests**: `parser/*_test.go` - AST validation
- **Integration**: Through VM execution tests

### Dependencies (Not Obvious)

- **go-git**: Used for Git-based package registries
- **go-billy**: Virtual filesystem for package management
- **google/go-cmp**: Test comparison utilities
- **Native Go**: No external build tools, pure Go implementation

### Standards & Style

- **Code Format**: Use `gofmt` - tabs for indentation, Go standard formatting
- **Testing**: Table-driven tests, helper functions for common setup
- **Naming**: Go conventions - exported/unexported based on capital letters
- **Documentation**: Godoc comments for public APIs

### Validation Checklist for Changes

1. **Format**: `task fmt`
2. **Check**: `task check`
3. **Race**: `go test -race ./...` for concurrency-related changes

### Common Gotchas & Workarounds

- **Formatting**: Several files have tab/space mix - always use `gofmt -w` to fix
- **Module cache**: If weird dependency errors, try `go clean -cache && go mod tidy`
- **Test timing**: `gitreg` tests can be slow (~800ms) due to Git operations
- **CLI entrypoint**: `cmd/zirric/main.go` is the executable entry for the CLI

## Zirric Evolution Proposals (ZE)

Proposals live in `docs/proposals/` and follow a defined lifecycle. Each proposal has a status: **Draft**, **In Progress**, **Implemented**, or **Rejected**.

### Creating a New Proposal

1. Copy `docs/proposals/ZE-000-template.md` to `docs/proposals/ZE-NNN-short-name.md` where `NNN` is the next available number.
2. Update the frontmatter (`title`, `description`) and fill out all template sections.
3. Keep exactly one status callout block from the template (Draft for new proposals) and remove the others.
4. Add a row to the table in `docs/proposals/index.md` with the correct status.
5. Add a navigation entry in `tasks/docmd/docmd.config.js` under the `Proposals` children array. Use the appropriate icon for the status:
   - Draft → `icon: "circle"`
   - In Progress → `icon: "circle-dot"`
   - Implemented → `icon: "circle-check"`
   - Rejected → `icon: "circle-x"`

### Updating Proposal Status

When changing a proposal's status, update **all three** locations:

1. **The proposal file** (`docs/proposals/ZE-NNN-*.md`): Replace the status callout block with the appropriate one from the template (`ZE-000-template.md`). Each status has a distinct callout style and SVG icon:
   - Draft: `::: callout draft` with circle SVG
   - In Progress: `::: callout warning` with circle-dot SVG
   - Implemented: `::: callout tip` with circle-check SVG
   - Rejected: `::: callout danger` with circle-x SVG
2. **The index** (`docs/proposals/index.md`): Update the Status column in the proposals table.
3. **The navigation config** (`tasks/docmd/docmd.config.js`): Update the `icon` field of the corresponding entry in the Proposals children array.

### When a Proposal Is Implemented

After marking a proposal as Implemented, perform these additional steps:

1. **Update the specification**: Reflect the new feature in the relevant files under `docs/specification/` (syntax, declarations, expressions, typesystem). See [Keeping the Specification Up to Date](#keeping-the-specification-up-to-date) below.
2. **Update the getting-started guide**: If the feature affects onboarding or common usage, update `docs/guides/getting-started.md`.
3. **Update the changelog**: Add the proposal to the current release notes file under `docs/changelog/` (see [Maintaining the Changelog](#maintaining-the-changelog) below).
4. **Search for outdated code and docs**: Look for old APIs, syntax, or descriptions that contradict the implemented proposal in:
   - `*.zirr` source files (e.g., `prelude/`, `future/`, `examples/`)
   - `Cavefile` files (e.g., `examples/project/Cavefile`)
   - All documentation under `docs/` (excluding other proposals in `docs/proposals/`)
5. **Do NOT update other proposals** in `docs/proposals/` — proposals are historical records of their time. Update all other outdated documentation under `docs/`.

## Keeping the Specification Up to Date

The specification (`docs/specification/`) documents the **actual** state of the language as implemented in code. Proposals (`docs/proposals/`) document a **desired** state that may be incomplete, outdated, or not yet implemented. The specification is the authoritative reference — proposals are historical records.

### When to update the specification

Update the specification whenever a code change affects what the language accepts or how it behaves:

- **New syntax or keywords**: Update `docs/specification/syntax.md` (grammar) and the relevant semantic page.
- **Changed semantics** (scoping, evaluation, matching): Update the semantic page (declarations, expressions, or typesystem).
- **New or changed type behavior**: Update `docs/specification/typesystem.md`.
- **New or changed declarations**: Update `docs/specification/declarations.md`.
- **Changed control flow or operators**: Update `docs/specification/expressions.md`.

If a change only affects internal compiler implementation without changing observable language behavior, the specification does not need updating.

### Specification files

| File              | Covers                                                                     |
| ----------------- | -------------------------------------------------------------------------- |
| `syntax.md`       | Formal grammar (EBNF), tokens, keywords, operator precedence               |
| `declarations.md` | All declaration forms, scoping, attribute application, type hint positions |
| `expressions.md`  | Expressions, operators, control flow, closures, switch/is, assignment      |
| `typesystem.md`   | Type categories, union membership, type hints, is-matching, protocols      |
| `index.md`        | Navigation guide with quick-reference table                                |

### Rules

- **No proposal references** in the specification. The specification stands on its own.
- **Cross-reference** between specification pages using relative links (e.g., `[Expressions § Switch](/specification/expressions#switch)`).
- **Keep in sync with code.** If a PR changes parser, compiler, or runtime behavior, the specification must be updated in the same PR or a follow-up.

## Maintaining the Changelog

The changelog lives under `docs/changelog/`. It consists of:

- **`index.md`** — Overview page using the [docmd changelog container syntax](https://docs.docmd.io/content/containers/changelogs/). Each release gets a one-line summary and a link to its full release notes page.
- **One file per release** (e.g., `v0.1.0.md`, `v0.0.1.md`) — Full release notes with all details. Each file must include YAML frontmatter with `title` (the version, use a `-next` suffix while unreleased) and `description` (a one-line summary of the release). The file name always matches the final release tag (no `-next` suffix) so that the URL remains stable across prerelease and release.

The current (unreleased) release uses its target version as file name (e.g., `v0.1.0.md`) but keeps a `-next` suffix in its `title` frontmatter and heading (e.g., `v0.1.0-next`). It also includes a prerelease callout. When the release is tagged, remove the `-next` suffix from the title, heading, and nav entry, remove the prerelease callout, and create a new file for the next cycle. Update `index.md` and the navigation in `tasks/docmd/docmd.config.js`.

The GitHub Actions release workflow (`.github/workflows/goreleaser.yaml`) automatically derives the release notes file from the Git tag (e.g., tag `v0.1.0` → `docs/changelog/v0.1.0.md`) and passes it to GoReleaser via `--release-notes`. This is why the file name must match the tag exactly.

### Headings in Release Notes

Use the following headings within each release notes file. Only include a heading if there are items for it.

- **`## Breaking Changes`** — Any change that breaks existing Zirric code or requires user migration (renamed keywords, changed syntax, removed features).
- **`## Implemented Proposals`** — Proposals that have been marked as Implemented. Link to the proposal page using `[ZE-NNN Title](/proposals/ZE-NNN-slug)` and add a brief one-line summary.
- **`## Notable Changes`** — Significant improvements, new capabilities, or behavioral changes that are not tied to a specific proposal.
- **`## Bug Fixes`** — Fixed bugs that are not already covered by an implemented proposal entry. One line per fix with a brief description.
- **`## Tooling`** — Changes to editor support, Tree-Sitter grammars, CLI tooling, or development infrastructure.

### Rules

- When a proposal is marked as Implemented, always add it under `## Implemented Proposals` in the current release notes file.
- Breaking changes always get their own entry under `## Breaking Changes`, even if they are also part of a proposal.
- Bug fixes that are part of an implemented proposal do not need a separate `## Bug Fixes` entry.
- Bug fixes unrelated to any proposal should be listed under `## Bug Fixes`.

## Instructions for Coding Agents

**Trust these instructions** - they are comprehensive and tested. Only search/explore if:

1. These instructions contradict current code structure
2. New build failures occur that aren't covered here
3. Instructions appear incomplete for your specific task

**Always run the validation checklist before finalizing changes.** The GitHub Actions CI will reject PRs that fail formatting, building, or testing.

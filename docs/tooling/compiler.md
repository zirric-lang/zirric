---
title: Compiler
description: High-level overview of the Zirric bytecode compiler and virtual machine.
---

# Compiler

This document describes the bytecode compiler and virtual machine at a high level for language tooling and implementation work. It is intentionally brief and the Go implementation is the source of truth.

## Pipeline overview

1. **Parsing**: source is tokenized and parsed into an AST.
2. **Analysis**: an analyzer resolves symbols, validates static references, and assigns IDs.
3. **Compilation**: the compiler emits bytecode instructions and a constant pool.
4. **Execution**: the VM evaluates bytecode with a stack-based model.

## Bytecode model

- **Constant pool**: literals and compiled functions live in a constant pool referenced by index.
- **Instruction stream**: opcodes are encoded in big endian with optional operands.
- **Stack**: most instructions pop operands and push results.
- **Control flow**: conditional and unconditional jumps alter the instruction pointer.

## Functions and closures

Functions compile to bytecode chunks with metadata such as arity. Closures capture their environment using upvalue cells — heap-allocated boxes that allow mutable `var` bindings to be shared between the enclosing scope and all capturing closures. The `MakeClosure` opcode builds a closure from a compiled function and its captured values, while `WrapLocal` converts a local into an upvalue cell. Closures are invoked through the `Call` opcode in the VM.

## Iteration

`for element <- value` compiles to a call into the value's `@Iterable` implementation. `MakeIterYield` builds a closure over the loop body — it carries the binding's local slot and the body's instruction range — and `CallIterate` invokes the iterable with the value and that closure. A `return` inside the loop body has to escape both the closure and the iterable that called it, so the VM raises it as a signal and unwinds to the frame the loop belongs to — a custom iterable behaves like a built-in one.

## Type hints and type matching

Type hints (`: T`, `-> T`) are parsed into `TypeExpr` AST nodes and stored on declarations and parameters. The `IsType` opcode performs runtime type checking against data types, extern types, union membership, and attribute constraints. `switch` statements compile type patterns (`case is T:`) using `IsType`.

## Guarded member access

All four begin with an `asoption` or `asresult` over the target. A value already in the union it names is left alone; anything else is asked for one through the `toOption` or `toResult` of its attribute, which the VM calls reentrantly the same way an extern function calls back into Zirric. What is on the stack afterwards is therefore always a `Some`, `None`, `Ok` or `Err`, so the instructions that follow only have to tell those apart, and the `getfield "value"` that ends each of them is the unwrapping step nobody writes.

`?.` then emits a `jumpistype` against `None` that leaves the value where it is, so the jump carries the `None` itself to the end of the chain; every `?.` in one chain jumps to the same place, where a `wrapoption` turns the value the chain did produce into an `Option`. Because the chain is compiled as one unit, a call cannot be part of it — the call's arguments are already on the stack when the short-circuit would jump past it — which is why the compiler rejects that shape instead.

`!.` emits a `returnistype` against `Err`, which returns the error from the running frame. Nothing has to be unwound or jumped over, since the frame is discarded along with whatever else is on its stack.

`??` and `!!` are the same shape as each other: one `jumpistype` over the normalized left operand — `None` for `??`, `Err` for `!!` — and a `pop` in front of the right operand, which is only ever reached on the missing path.

## Core opcodes

| Mnemonic        | Widths | Description                                     | Comments                            |
| --------------- | ------ | ----------------------------------------------- | ----------------------------------- |
| `const`         | 2      | Push constant from constant pool                |                                     |
| `constvoid`     | 0      | Push `void`                                     |                                     |
| `consttrue`     | 0      | Push boolean `true`                             |                                     |
| `constfalse`    | 0      | Push boolean `false`                            |                                     |
| `pop`           | 0      | Discard top of stack                            |                                     |
| `array`         | 0      | Build array from preceding values               | length on stack                     |
| `dict`          | 0      | Build dictionary from preceding key/value pairs | length on stack                     |
| `module`        | 2      | Push module value                               |                                     |
| `getindex`      | 0      | Index into array or dict                        | key/index and collection on stack   |
| `getfield`      | 2      | Access field by name                            |                                     |
| `setfield`      | 2      | Set field by name                               |                                     |
| `setindex`      | 0      | Set value at index/key                          | value, key, collection on stack     |
| `len`           | 0      | Push length of collection                       |                                     |
| `arrayappend`   | 0      | Append value to array                           |                                     |
| `asserttype`    | 2      | Assert top value has given type ID              |                                     |
| `istype`        | 2      | Check if value is of type; push Bool            | union membership supported          |
| `asoption`      | 2,2    | Read the top value as an Option                 | Option union, @AnyOption attribute  |
| `asresult`      | 2,2    | Read the top value as a Result                  | Result union, @AnyResult attribute  |
| `jumpistype`    | 2,2    | Jump if top value is of type, leaving it        | address, type constant              |
| `returnistype`  | 2      | Return top value if it is of type               | type constant                       |
| `wrapoption`    | 2,2    | Wrap top value in `Some` unless already Option  | Option union, `Some` data type      |
| `jump`          | 2      | Unconditional jump to address                   |                                     |
| `jumptrue`      | 2      | Jump if top value is truthy                     |                                     |
| `jumpfalse`     | 2      | Jump if top value is `false`                    |                                     |
| `negate`        | 0      | Numeric negation                                |                                     |
| `invert`        | 0      | Boolean NOT                                     |                                     |
| `add`           | 0      | Add two numbers                                 |                                     |
| `sub`           | 0      | Subtract two numbers                            |                                     |
| `mul`           | 0      | Multiply two numbers                            |                                     |
| `div`           | 0      | Divide two numbers                              |                                     |
| `mod`           | 0      | Remainder of integer division                   |                                     |
| `eq`            | 0      | Compare for equality                            |                                     |
| `neq`           | 0      | Compare for inequality                          |                                     |
| `gt`            | 0      | Compare greater-than                            |                                     |
| `gte`           | 0      | Compare greater-than-or-equal                   |                                     |
| `lt`            | 0      | Compare less-than                               |                                     |
| `lte`           | 0      | Compare less-than-or-equal                      |                                     |
| `makeattribute` | 2      | Create attribute instance                       |                                     |
| `call`          | 0      | Call function or closure                        | arg count on stack                  |
| `makeiteryield` | 2,2,2  | Build the yield closure of a `for <-` loop      | binding local, body start, body end |
| `calliterate`   | 2      | Drive `@Iterable` with a value and that closure | arg count, always 2                 |
| `return`        | 0      | Return from function                            |                                     |
| `getglobal`     | 2      | Push global variable                            |                                     |
| `setglobal`     | 2      | Set global variable                             |                                     |
| `getlocal`      | 1      | Push local variable                             |                                     |
| `setlocal`      | 1      | Set local variable                              |                                     |
| `makeclosure`   | 2, 1   | Create closure from compiled function           | const ID + free count               |
| `getfree`       | 1      | Push captured value (const capture)             |                                     |
| `getfreecell`   | 1      | Read captured var via upvalue cell              |                                     |
| `setfreecell`   | 1      | Write captured var via upvalue cell             |                                     |
| `getlocalcell`  | 1      | Read local var through upvalue cell             |                                     |
| `setlocalcell`  | 1      | Write local var through upvalue cell            |                                     |
| `wraplocal`     | 1      | Wrap local in upvalue cell                      | emitted for captured vars           |
| `debug`         | 0      | Optional breakpoint instruction                 | omitted in release builds           |

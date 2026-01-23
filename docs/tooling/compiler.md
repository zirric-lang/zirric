---
title: Compiler
description: High-level overview of the Zirric bytecode compiler and virtual machine.
---

# Compiler

This document describes the bytecode compiler and virtual machine at a high
level for language tooling and implementation work. It is intentionally brief;
the Go implementation is the source of truth.

## Pipeline overview

1. **Parsing**: source is tokenized and parsed into an AST.
2. **Compilation**: the compiler emits bytecode instructions and a constant pool.
3. **Execution**: the VM evaluates bytecode with a stack-based model.

## Bytecode model

- **Constant pool**: literals and compiled functions live in a constant pool
  referenced by index.
- **Instruction stream**: opcodes are encoded in big endian with optional
  operands.
- **Stack**: most instructions pop operands and push results.
- **Control flow**: conditional and unconditional jumps alter the instruction
  pointer.

## Functions and closures

Functions compile to bytecode chunks with metadata such as arity. Closures carry
their captured environment and are invoked through the `call` opcode in the VM.

## Core opcodes

| Mnemonic   | Widths | Description                                     | Comments                  |
| ---------- | ------ | ----------------------------------------------- | ------------------------- |
| const      | 2      | Push constant from constant pool                |                           |
| consttrue  | 0      | Push boolean `true`                             |                           |
| constfalse | 0      | Push boolean `false`                            |                           |
| pop        | 0      | Discard top of stack                            |                           |
| array      | 0      | Build array from preceding values               | length on stack           |
| dict       | 0      | Build dictionary from preceding key/value pairs | length on stack           |
| asserttype | 2      | Assert top value has given type ID              |                           |
| jump       | 2      | Unconditional jump to address                   |                           |
| jumptrue   | 2      | Jump if top value is truthy                     |                           |
| jumpfalse  | 2      | Jump if top value is `false`                    |                           |
| negate     | 0      | Numeric negation                                |                           |
| invert     | 0      | Boolean NOT                                     |                           |
| add        | 0      | Add two numbers                                 |                           |
| sub        | 0      | Subtract two numbers                            |                           |
| mul        | 0      | Multiply two numbers                            |                           |
| div        | 0      | Divide two numbers                              |                           |
| mod        | 0      | Remainder of integer division                   |                           |
| eq         | 0      | Compare for equality                            |                           |
| neq        | 0      | Compare for inequality                          |                           |
| gt         | 0      | Compare greater-than                            |                           |
| gte        | 0      | Compare greater-than-or-equal                   |                           |
| lt         | 0      | Compare less-than                               |                           |
| lte        | 0      | Compare less-than-or-equal                      |                           |
| debug      | 0      | Optional breakpoint instruction                 | omitted in release builds |

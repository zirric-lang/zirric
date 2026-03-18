# Syntax

This page defines the formal grammar of Zirric. It covers every syntactic form the parser accepts. For the meaning of each construct, see the corresponding semantic pages: [Declarations](/specification/declarations), [Expressions](/specification/expressions), and [Type System](/specification/typesystem).

The grammar is written in EBNF notation. Terminal symbols are quoted. `{x}` means zero or more repetitions of `x`, and `[x]` means `x` is optional.

## Tokens

### Identifiers

An identifier starts with a letter or underscore and continues with letters, digits, or underscores. Keywords take precedence over identifiers.

```ebnf
IDENT = (letter | "_"), {letter | digit | "_"};
```

### Keywords

The following words are reserved and cannot be used as identifiers:

```
mod    import   fn      const   var
data   union    extern  type    attr
if     else     for     switch  case
is     return   break   continue
true   false    _
```

### Literal Tokens

```ebnf
INT    = "0" | ("1"..."9"), {digit}
       | "0x", hex_digit, {hex_digit}
       | "0o", octal_digit, {octal_digit}
       | "0b", ("0" | "1"), {("0" | "1")};

FLOAT  = digits, ".", digits, [("e" | "E"), ["-"], digits]
       | digits, ("e" | "E"), ["-"], digits;

STRING = '"', {string_char}, '"';

BOOL   = "true" | "false";
```

### Operator and Punctuation Tokens

```ebnf
PLUS     = "+";     MINUS    = "-";     ASTERISK = "*";
SLASH    = "/";     PERCENT  = "%";     BANG     = "!";

EQ       = "==";    NEQ      = "!=";
LT       = "<";     GT       = ">";
LTE      = "<=";    GTE      = ">=";
AND      = "&&";    OR       = "||";

ASSIGN   = "=";     ARROW    = "->";    LARROW   = "<-";
COLON    = ":";     DOT      = ".";     COMMA    = ",";
AT       = "@";

LPAREN   = "(";     RPAREN   = ")";
LBRACE   = "{";     RBRACE   = "}";
LBRACKET = "[";     RBRACKET = "]";
```

### Comments

```ebnf
line_comment  = "//", {any_char}, newline;
block_comment = "/*", {any_char}, "*/";
```

Block comments may nest.

## Operator Precedence

Operators are listed from lowest to highest precedence.

| Precedence  | Operators                        | Associativity |
| ----------- | -------------------------------- | ------------- |
| Logical OR  | `\|\|`                           | Left          |
| Logical AND | `&&`                             | Left          |
| Comparison  | `==`, `!=`, `<`, `<=`, `>`, `>=` | None          |
| Sum         | `+`, `-`                         | Left          |
| Product     | `*`, `/`, `%`                    | Left          |
| Prefix      | `-x`, `!x`                       | Right         |
| Is          | `expr is Type`                   | Left          |
| Call        | `f(args)`                        | Left          |
| Member      | `expr.field`                     | Left          |

See [Expressions § Operators](/specification/expressions#operators) for semantic details.

## Source File Structure

A source file begins with an optional shebang line and module declaration, followed by top-level declarations.

```ebnf
SourceFile = [shebang], [Module], {TopLevelDeclaration};

shebang = "#!", {any_inline_char}, newline;

Module = "mod", Identifier;
```

## Declarations

See [Declarations](/specification/declarations) for semantic rules.

### Variable and Constant Declarations

```ebnf
Const = "const", Identifier, [":", TypeHint], "=", Expression;
Var   = "var",   Identifier, [":", TypeHint], "=", Expression;
```

### Function Declaration

```ebnf
Function = "fn", Identifier, "(", [Parameters], ")", ["->", TypeHint], "{", Block, "}";
```

### Data Declaration

```ebnf
Data = "data", Identifier, ["{", {DataField}, "}"];

DataField = {Attribute}, Identifier, [":", TypeHint], [","];
```

A `data` declaration without a body (no braces) declares a zero-field type.

### Union Declaration

```ebnf
Union = "union", Identifier, "{", {UnionMember}, "}";

UnionMember = {Attribute}, (Data | StaticReference), [","];
```

Union members are either inline `data` declarations or references to existing types. See [Type System § Union Types](/specification/typesystem#union-types) for membership semantics.

### Attribute Declaration

```ebnf
AttributeDecl = "attr", Identifier, ["{", {DataField}, "}"];
```

Attribute declarations follow the same field syntax as `data`. See [Declarations § Attribute Declarations](/specification/declarations#attr) for application rules.

### Extern Declarations

```ebnf
ExternType  = "extern", "type", Identifier, ["{", {DataField}, "}"];
ExternFunc  = "extern", "fn",   Identifier, "(", [Parameters], ")";
ExternConst = "extern", "const", Identifier, [":", TypeHint];
```

### Import Declaration

```ebnf
Import = "import", [Identifier, "="], ImportPath, ["{", {Identifier, [","]}, "}"];

ImportPath = Identifier, {".", Identifier};
```

### Attribute Application

```ebnf
Attribute = "@", StaticReference, "(", [Arguments], ")";
```

Attributes precede the declaration or field they annotate. Parentheses are always required, even when empty. Multiple attributes can be stacked.

```ebnf
TopLevelDeclaration = Import
                    | {Attribute}, (Union | Data | ExternType | ExternFunc | ExternConst | AttributeDecl | Function | Const | Var)
                    | Expression;
```

## Parameters

```ebnf
Parameters = Parameter, {",", Parameter};
Parameter  = Identifier, [":", TypeHint];
```

Parameters appear in `fn` declarations, closures, `extern fn`, and function-style data fields. See [Declarations § Parameters](/specification/declarations#parameters-and-type-hints).

## Type Hints

Type hints annotate declarations and parameters with type information. They appear after `:` on fields, parameters, and variables, and after `->` on function return types.

```ebnf
TypeHint     = TypeHintRef | TypeHintArray | TypeHintDict | TypeHintFunc | TypeHintAttrs;

TypeHintRef  = StaticReference;
TypeHintArray = "[", TypeHint, "]";
TypeHintDict = "[", TypeHint, ":", TypeHint, "]";
TypeHintFunc = "fn", "(", [TypeHintParams], ")", ["->", TypeHint];
TypeHintAttrs = "@", StaticReference, {"@", StaticReference};
```

See [Type System § Type Hints](/specification/typesystem#type-hints) for what type hints express and how they interact with type checking.

## Expressions

See [Expressions](/specification/expressions) for semantic details.

### Literals

```ebnf
IntLiteral    = INT;
FloatLiteral  = FLOAT;
StringLiteral = STRING;
BoolLiteral   = "true" | "false";
ArrayLiteral  = "[", [Expression, {",", Expression}], "]";
DictLiteral   = "[", [DictEntry, {",", DictEntry}], "]";

DictEntry = Expression, ":", Expression;
```

### Identifiers and References

```ebnf
Identifier      = IDENT;
StaticReference = Identifier, {".", Identifier};
```

A `StaticReference` is a dot-separated path used for qualified names in imports, attributes, and type hints.

### Operators

```ebnf
InfixExpr  = Expression, operator, Expression;
PrefixExpr = ("-" | "!"), Expression;
IsExpr     = Expression, "is", TypeHint;
```

### Calls and Member Access

```ebnf
CallExpr   = Expression, "(", [Arguments], ")";
MemberExpr = Expression, ".", Identifier;
IndexExpr  = Expression, "[", Expression, "]";

Arguments = Expression, {",", Expression};
```

### Closures

```ebnf
Closure = "fn", "(", [Parameters], ")", ["->", TypeHint], "{", Block, "}";
```

Closures are anonymous functions. They share syntax with `fn` declarations but omit the name. See [Expressions § Closures](/specification/expressions#closures).

### Switch Expression

```ebnf
SwitchExpr = "switch", Expression, "{", {SwitchCase}, "}";
SwitchCase = "case", (["is"], CasePattern), ":", Expression;

CasePattern = TypeHint | Expression | "_";
```

Switch expressions require a `_` fallback case. See [Expressions § Switch](/specification/expressions#switch).

### Assignment

```ebnf
Assignment      = Identifier, "=", Expression;
FieldAssignment = Expression, ".", Identifier, "=", Expression;
IndexAssignment = Expression, "[", Expression, "]", "=", Expression;
```

Only `var` bindings and mutable locations accept assignment. See [Expressions § Assignment](/specification/expressions#assignment).

## Statements

Statements appear inside blocks. Most expressions can also be used as statements.

```ebnf
Block = {Statement};

Statement = ScopeLevelDeclaration | Return | If | For | SwitchStmt | Expression;

ScopeLevelDeclaration = Function | Const | Var | Union | Data;
```

### Return

```ebnf
Return = "return", [Expression];
```

### If

```ebnf
If = "if", Expression, "{", Block, "}", {"else", "if", Expression, "{", Block, "}"}, ["else", "{", Block, "}"];
```

See [Expressions § If](/specification/expressions#if) for the distinction between `if` expressions and `if` statements.

### For

```ebnf
For = "for", [ForBinding], "{", Block, "}";

ForBinding = Identifier, "<-", Expression    (* collection iteration *)
           | Expression;                      (* boolean or infinite loop *)
```

When the binding and expression are both omitted, the loop runs indefinitely. See [Expressions § For](/specification/expressions#for).

### Switch Statement

```ebnf
SwitchStmt     = "switch", Expression, "{", {StmtSwitchCase}, "}";
StmtSwitchCase = "case", (["is"], CasePattern), ":", Block;
```

Switch statements do not require a `_` fallback.

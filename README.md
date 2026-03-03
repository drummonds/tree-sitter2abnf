# tree-sitter2abnf

Converts tree-sitter `grammar.json` files to extended ABNF (RFC 5234) and back.

## Install

```
go install github.com/drummonds/tree-sitter2abnf/cmd/tree-sitter2abnf@latest
```

## Usage

```
tree-sitter2abnf [flags] <input-file>
```

Direction is auto-detected from file extension:
- `.json` → ABNF output
- `.abnf` → JSON output

Flags:
- `-o FILE` — output file (default: stdout)
- `-v` — show version

### Examples

```sh
# Convert grammar.json to ABNF
tree-sitter2abnf grammar.json -o grammar.abnf

# Convert ABNF back to grammar.json
tree-sitter2abnf grammar.abnf -o grammar.json
```

## ABNF Extension Syntax

Standard ABNF (RFC 5234 + RFC 7405) covers concatenation, alternation, repetition, optional groups, case-sensitive strings, and value ranges. This tool extends ABNF with annotations to represent tree-sitter concepts.

### Direct Mappings

| Tree-sitter | ABNF |
|---|---|
| `SEQ(a, b)` | `a b` (concatenation) |
| `CHOICE(a, b)` | `(a / b)` (alternation) |
| `REPEAT(x)` | `*x` |
| `REPEAT1(x)` | `1*x` |
| `STRING("lit")` | `%s"lit"` |
| `SYMBOL(name)` | `name` |
| `CHOICE(x, BLANK)` | `[x]` (optional) |

### Rule-level Annotations

| Tree-sitter | Extended ABNF |
|---|---|
| `PREC(N, x)` | `@prec(N) x` |
| `PREC_LEFT(N, x)` | `@prec-left(N) x` |
| `PREC_RIGHT(N, x)` | `@prec-right(N) x` |
| `PREC_DYNAMIC(N, x)` | `@prec-dynamic(N) x` |
| `FIELD("name", x)` | `@field(name) x` |
| `ALIAS(x, "n")` | `@alias(n) x` |
| `ALIAS(x, "n", named)` | `@alias(~n) x` |
| `TOKEN(x)` | `@token(x)` |
| `IMMEDIATE_TOKEN(x)` | `@immediate-token(x)` |
| `PATTERN("re")` | `@pattern("re")` |

### Grammar-level Directives

Emitted as ABNF comments to preserve validity:

```abnf
; @grammar "json"
; @word "identifier"
; @extras (@pattern("\\s") / comment)
; @inline (_semicolon / _call_signature)
; @conflicts (call_expression member_expression)
; @externals (_ternary_qmark / _template_chars)
; @supertypes (expression / statement)
```

### Rule Naming

Underscores are allowed in rule names (extension to RFC 5234). Hidden rules (tree-sitter `_`-prefixed) keep their underscore prefix.

## Example Output

Given the tree-sitter-json `grammar.json`:

```abnf
; @grammar "json"
; @extras (@pattern("\\s") / comment)
; @supertypes (_value)

document = *_value

_value = (object / array / number / string / true / false / null)

object = %s"{" [pair *(%s"," pair)] %s"}"

pair = @field(key) string %s":" @field(value) _value
```

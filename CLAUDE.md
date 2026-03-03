# tree-sitter2abnf

Converts tree-sitter `grammar.json` to extended ABNF and back.

## Key files
- `internal/grammar/grammar.go` — Grammar JSON types with ordered-key unmarshal
- `internal/abnf/writer.go` — Grammar → ABNF text
- `internal/abnf/parser.go` — ABNF text → Grammar
- `internal/abnf/extensions.go` — Extension directive constants
- `cmd/tree-sitter2abnf/main.go` — CLI entry point

## Testing
- `task check` runs fmt + vet + test
- Golden files in `testdata/abnf/` — update with `go test ./internal/abnf/ -update`
- Round-trip tests verify `json → abnf → json` structural equality

## ABNF extensions
Uses `@directive` annotations in ABNF comments and inline to represent tree-sitter
concepts not expressible in standard ABNF. See README.md for full reference.

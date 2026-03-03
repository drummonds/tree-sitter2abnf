# Roadmap

## v0.1.0
- [x] grammar.json types with ordered-key JSON unmarshal
- [x] JSON → ABNF writer with all extensions
- [x] ABNF → JSON parser (recursive descent)
- [x] Round-trip tests
- [x] CLI with auto-detection

## Future
- Support `precedences` (named precedence levels)
- Support `reserved` keyword sets (newer tree-sitter feature)
- ABNF validation / linting mode
- Diff mode: compare two grammars via their ABNF representation

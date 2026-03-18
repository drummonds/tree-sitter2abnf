// Package abnf handles conversion between tree-sitter grammar.json and extended ABNF.
package abnf

import "strings"

// Grammar-level directive names (emitted as comments).
const (
	DirGrammar    = "@grammar"
	DirWord       = "@word"
	DirExtras     = "@extras"
	DirInline     = "@inline"
	DirConflicts  = "@conflicts"
	DirExternals  = "@externals"
	DirSupertypes = "@supertypes"
)

// Rule-level inline annotation names.
const (
	AnnPrec        = "@prec"
	AnnPrecLeft    = "@prec-left"
	AnnPrecRight   = "@prec-right"
	AnnPrecDynamic = "@prec-dynamic"
	AnnField       = "@field"
	AnnAlias       = "@alias"
	AnnToken       = "@token"
	AnnImmToken    = "@immediate-token"
	AnnPattern     = "@pattern"
)

// ToABNFName converts a tree-sitter rule name to an ABNF-compatible name
// by replacing non-leading underscores with hyphens.
// The leading underscore (hidden-rule marker) is preserved.
func ToABNFName(name string) string {
	if strings.HasPrefix(name, "_") {
		return "_" + strings.ReplaceAll(name[1:], "_", "-")
	}
	return strings.ReplaceAll(name, "_", "-")
}

// ToGrammarName converts an ABNF rule name back to a tree-sitter name
// by replacing non-leading hyphens with underscores.
func ToGrammarName(name string) string {
	if strings.HasPrefix(name, "_") {
		return "_" + strings.ReplaceAll(name[1:], "-", "_")
	}
	return strings.ReplaceAll(name, "-", "_")
}

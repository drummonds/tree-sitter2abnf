// Package abnf handles conversion between tree-sitter grammar.json and extended ABNF.
package abnf

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

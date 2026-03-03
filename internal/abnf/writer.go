package abnf

import (
	"fmt"
	"strings"

	"github.com/drummonds/tree-sitter2abnf/internal/grammar"
)

// Write converts a Grammar to extended ABNF text.
func Write(g *grammar.Grammar) string {
	var b strings.Builder

	// Grammar-level directives
	writeDirectives(&b, g)

	if len(g.Rules) > 0 {
		b.WriteString("\n")
	}

	// Rules
	for i, nr := range g.Rules {
		if i > 0 {
			b.WriteString("\n")
		}
		writeNamedRule(&b, nr)
	}

	return b.String()
}

func writeDirectives(b *strings.Builder, g *grammar.Grammar) {
	fmt.Fprintf(b, "; %s %q\n", DirGrammar, g.Name)

	if g.Word != "" {
		fmt.Fprintf(b, "; %s %q\n", DirWord, g.Word)
	}

	if len(g.Extras) > 0 {
		fmt.Fprintf(b, "; %s (", DirExtras)
		for i, e := range g.Extras {
			if i > 0 {
				b.WriteString(" / ")
			}
			b.WriteString(writeRuleInline(e))
		}
		b.WriteString(")\n")
	}

	if len(g.Inline) > 0 {
		fmt.Fprintf(b, "; %s (", DirInline)
		for i, name := range g.Inline {
			if i > 0 {
				b.WriteString(" / ")
			}
			b.WriteString(name)
		}
		b.WriteString(")\n")
	}

	if len(g.Conflicts) > 0 {
		fmt.Fprintf(b, "; %s", DirConflicts)
		for _, group := range g.Conflicts {
			b.WriteString(" (")
			b.WriteString(strings.Join(group, " "))
			b.WriteString(")")
		}
		b.WriteString("\n")
	}

	if len(g.Externals) > 0 {
		fmt.Fprintf(b, "; %s (", DirExternals)
		for i, e := range g.Externals {
			if i > 0 {
				b.WriteString(" / ")
			}
			b.WriteString(writeRuleInline(e))
		}
		b.WriteString(")\n")
	}

	if len(g.Supertypes) > 0 {
		fmt.Fprintf(b, "; %s (", DirSupertypes)
		for i, name := range g.Supertypes {
			if i > 0 {
				b.WriteString(" / ")
			}
			b.WriteString(name)
		}
		b.WriteString(")\n")
	}
}

func writeNamedRule(b *strings.Builder, nr grammar.NamedRule) {
	rhs := writeRuleExpr(nr.Rule, false)
	// Use =/ for multi-line alternatives if the line would be too long
	line := fmt.Sprintf("%s = %s", nr.Name, rhs)
	if len(line) <= 120 {
		b.WriteString(line)
		b.WriteString("\n")
		return
	}

	// Long rule: break alternatives onto separate lines
	if nr.Rule.Type == grammar.TypeCHOICE {
		fmt.Fprintf(b, "%s =\n", nr.Name)
		for i, m := range nr.Rule.Members {
			prefix := "    / "
			if i == 0 {
				prefix = "      "
			}
			b.WriteString(prefix)
			b.WriteString(writeRuleExpr(m, false))
			b.WriteString("\n")
		}
		return
	}

	b.WriteString(line)
	b.WriteString("\n")
}

// writeRuleExpr returns the ABNF expression for a rule node.
// inGroup indicates whether we're inside a group (affects parenthesization).
func writeRuleExpr(r grammar.Rule, inGroup bool) string {
	switch r.Type {
	case grammar.TypeBLANK:
		// BLANK shouldn't appear standalone normally; if it does, empty string
		return "\"\""

	case grammar.TypeSTRING:
		return fmt.Sprintf("%%s%q", r.StringValue())

	case grammar.TypePATTERN:
		return fmt.Sprintf("%s(%q)", AnnPattern, r.StringValue())

	case grammar.TypeSYMBOL:
		return r.Name

	case grammar.TypeSEQ:
		parts := make([]string, len(r.Members))
		for i, m := range r.Members {
			s := writeRuleExpr(m, false)
			// Parenthesize nested SEQ to preserve structure on round-trip
			if m.Type == grammar.TypeSEQ {
				s = "(" + s + ")"
			}
			parts[i] = s
		}
		return strings.Join(parts, " ")

	case grammar.TypeCHOICE:
		// Check for optional: CHOICE(x, BLANK) or CHOICE(BLANK, x)
		if opt, ok := asOptional(r); ok {
			inner := writeRuleExpr(opt, true)
			return "[" + inner + "]"
		}
		parts := make([]string, len(r.Members))
		for i, m := range r.Members {
			parts[i] = writeRuleExpr(m, false)
		}
		s := strings.Join(parts, " / ")
		// Parenthesize if we're nested in SEQ context
		if !inGroup {
			return "(" + s + ")"
		}
		return s

	case grammar.TypeREPEAT:
		inner := writeRuleExpr(*r.Content, false)
		if needsParens(*r.Content) {
			return "*(" + inner + ")"
		}
		return "*" + inner

	case grammar.TypeREPEAT1:
		inner := writeRuleExpr(*r.Content, false)
		if needsParens(*r.Content) {
			return "1*(" + inner + ")"
		}
		return "1*" + inner

	case grammar.TypePREC:
		return writePrecExpr(AnnPrec, r)

	case grammar.TypePREC_LEFT:
		return writePrecExpr(AnnPrecLeft, r)

	case grammar.TypePREC_RIGHT:
		return writePrecExpr(AnnPrecRight, r)

	case grammar.TypePREC_DYNAMIC:
		return writePrecExpr(AnnPrecDynamic, r)

	case grammar.TypeFIELD:
		inner := writeRuleExpr(*r.Content, false)
		return fmt.Sprintf("%s(%s) %s", AnnField, r.Name, inner)

	case grammar.TypeALIAS:
		inner := writeRuleExpr(*r.Content, false)
		val := r.StringValue()
		if r.Named != nil && *r.Named {
			return fmt.Sprintf("%s(~%s) %s", AnnAlias, val, inner)
		}
		return fmt.Sprintf("%s(%s) %s", AnnAlias, val, inner)

	case grammar.TypeTOKEN:
		inner := writeRuleExpr(*r.Content, true)
		return fmt.Sprintf("%s(%s)", AnnToken, inner)

	case grammar.TypeIMMEDIATE_TOKEN:
		inner := writeRuleExpr(*r.Content, true)
		return fmt.Sprintf("%s(%s)", AnnImmToken, inner)

	default:
		return fmt.Sprintf("; UNKNOWN(%s)", r.Type)
	}
}

// writeRuleInline writes a compact inline form for use in directives.
func writeRuleInline(r grammar.Rule) string {
	return writeRuleExpr(r, false)
}

func writePrecExpr(ann string, r grammar.Rule) string {
	inner := writeRuleExpr(*r.Content, false)
	v := r.PrecValue()
	switch pv := v.(type) {
	case float64:
		return fmt.Sprintf("%s(%d) %s", ann, int(pv), inner)
	case string:
		return fmt.Sprintf("%s(%s) %s", ann, pv, inner)
	default:
		return fmt.Sprintf("%s(0) %s", ann, inner)
	}
}

// asOptional checks if a CHOICE is CHOICE(x, BLANK) or CHOICE(BLANK, x) with exactly 2 members.
func asOptional(r grammar.Rule) (grammar.Rule, bool) {
	if r.Type != grammar.TypeCHOICE || len(r.Members) != 2 {
		return grammar.Rule{}, false
	}
	if r.Members[1].Type == grammar.TypeBLANK {
		return r.Members[0], true
	}
	if r.Members[0].Type == grammar.TypeBLANK {
		return r.Members[1], true
	}
	return grammar.Rule{}, false
}

// needsParens returns true if the rule expression needs parenthesization
// when used as the argument to * or 1*.
func needsParens(r grammar.Rule) bool {
	switch r.Type {
	case grammar.TypeSEQ, grammar.TypeCHOICE:
		return true
	default:
		return false
	}
}

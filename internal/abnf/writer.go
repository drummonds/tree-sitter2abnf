package abnf

import (
	"fmt"
	"sort"
	"strings"

	"codeberg.org/hum3/tree-sitter2abnf/internal/grammar"
)

// Write converts a Grammar to extended ABNF text.
func Write(g *grammar.Grammar) string {
	var b strings.Builder

	// Grammar-level directives
	writeDirectives(&b, g)

	if len(g.Rules) == 0 {
		return b.String()
	}

	sorted, levels := sortRulesByComplexity(g.Rules)
	prevLevel := -1
	for i, nr := range sorted {
		level := levels[nr.Name]
		if prevLevel > 0 && level == 0 {
			b.WriteString("\n; --- Tokens ---\n\n")
		} else if i > 0 {
			b.WriteString("\n")
		} else {
			b.WriteString("\n")
		}
		prevLevel = level
		writeNamedRule(&b, nr)
	}

	return b.String()
}

func writeDirectives(b *strings.Builder, g *grammar.Grammar) {
	fmt.Fprintf(b, "; %s %q\n", DirGrammar, g.Name)

	if g.Word != "" {
		fmt.Fprintf(b, "; %s %q\n", DirWord, ToABNFName(g.Word))
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
			b.WriteString(ToABNFName(name))
		}
		b.WriteString(")\n")
	}

	if len(g.Conflicts) > 0 {
		fmt.Fprintf(b, "; %s", DirConflicts)
		for _, group := range g.Conflicts {
			b.WriteString(" (")
			for i, name := range group {
				if i > 0 {
					b.WriteString(" ")
				}
				b.WriteString(ToABNFName(name))
			}
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
			b.WriteString(ToABNFName(name))
		}
		b.WriteString(")\n")
	}
}

func writeNamedRule(b *strings.Builder, nr grammar.NamedRule) {
	name := ToABNFName(nr.Name)
	rhs := writeRuleExpr(nr.Rule, false)
	// Use =/ for multi-line alternatives if the line would be too long
	line := fmt.Sprintf("%s = %s", name, rhs)
	if len(line) <= 120 {
		b.WriteString(line)
		b.WriteString("\n")
		return
	}

	// Long rule: break alternatives onto separate lines
	if nr.Rule.Type == grammar.TypeCHOICE {
		fmt.Fprintf(b, "%s =\n", name)
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
		return ToABNFName(r.Name)

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
		if needsParens(*r.Content) {
			inner := writeRuleExpr(*r.Content, true)
			return "*(" + inner + ")"
		}
		return "*" + writeRuleExpr(*r.Content, false)

	case grammar.TypeREPEAT1:
		if needsParens(*r.Content) {
			inner := writeRuleExpr(*r.Content, true)
			return "1*(" + inner + ")"
		}
		return "1*" + writeRuleExpr(*r.Content, false)

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

// sortRulesByComplexity returns rules sorted with complex (structural) rules
// first and leaf (token) rules last. Within each level, original order is preserved.
func sortRulesByComplexity(rules []grammar.NamedRule) ([]grammar.NamedRule, map[string]int) {
	levels := computeRuleLevels(rules)
	sorted := make([]grammar.NamedRule, len(rules))
	copy(sorted, rules)
	origIdx := make(map[string]int, len(rules))
	for i, nr := range rules {
		origIdx[nr.Name] = i
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		li, lj := levels[sorted[i].Name], levels[sorted[j].Name]
		if li != lj {
			return li > lj
		}
		return origIdx[sorted[i].Name] < origIdx[sorted[j].Name]
	})
	return sorted, levels
}

// computeRuleLevels assigns each rule a dependency depth.
// Level 0 = leaf (no references to other grammar rules), higher = more complex.
func computeRuleLevels(rules []grammar.NamedRule) map[string]int {
	ruleSet := make(map[string]bool, len(rules))
	deps := make(map[string]map[string]bool, len(rules))
	for _, nr := range rules {
		ruleSet[nr.Name] = true
		deps[nr.Name] = collectSymbolRefs(nr.Rule)
	}
	// Keep only references to other rules in this grammar, excluding self-refs
	for name, refs := range deps {
		for ref := range refs {
			if !ruleSet[ref] || ref == name {
				delete(refs, ref)
			}
		}
	}

	levels := make(map[string]int, len(rules))
	remaining := make(map[string]bool, len(rules))
	for name := range ruleSet {
		remaining[name] = true
	}
	for level := 0; len(remaining) > 0; level++ {
		var resolved []string
		for name := range remaining {
			allDone := true
			for dep := range deps[name] {
				if remaining[dep] {
					allDone = false
					break
				}
			}
			if allDone {
				resolved = append(resolved, name)
			}
		}
		if len(resolved) == 0 {
			// Remaining rules form cycles — assign current level
			for name := range remaining {
				levels[name] = level
			}
			break
		}
		for _, name := range resolved {
			levels[name] = level
			delete(remaining, name)
		}
	}
	return levels
}

// collectSymbolRefs returns the set of SYMBOL names referenced by a rule tree.
func collectSymbolRefs(r grammar.Rule) map[string]bool {
	refs := make(map[string]bool)
	var walk func(grammar.Rule)
	walk = func(r grammar.Rule) {
		if r.Type == grammar.TypeSYMBOL {
			refs[r.Name] = true
			return
		}
		for _, m := range r.Members {
			walk(m)
		}
		if r.Content != nil {
			walk(*r.Content)
		}
	}
	walk(r)
	return refs
}

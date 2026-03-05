package abnf

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/drummonds/tree-sitter2abnf/internal/grammar"
)

// TestRoundTripMinimal tests json → abnf → json structural equality.
func TestRoundTripMinimal(t *testing.T) {
	testRoundTrip(t, "../../testdata/json/minimal.json")
}

// TestRoundTripJSON tests with the real tree-sitter-json grammar.
func TestRoundTripJSON(t *testing.T) {
	testRoundTrip(t, "../../testdata/json/json.json")
}

func testRoundTrip(t *testing.T, jsonPath string) {
	t.Helper()

	// Load original grammar
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	var original grammar.Grammar
	if err := json.Unmarshal(data, &original); err != nil {
		t.Fatal(err)
	}

	// Convert to ABNF
	abnfText := Write(&original)

	// Parse ABNF back
	roundTripped, err := Parse(abnfText)
	if err != nil {
		t.Fatalf("parse ABNF: %v\n\nABNF text:\n%s", err, abnfText)
	}

	// Compare structurally
	assertGrammarEqual(t, &original, roundTripped)
}

func assertGrammarEqual(t *testing.T, a, b *grammar.Grammar) {
	t.Helper()

	if a.Name != b.Name {
		t.Errorf("name: %q vs %q", a.Name, b.Name)
	}
	if a.Word != b.Word {
		t.Errorf("word: %q vs %q", a.Word, b.Word)
	}
	if len(a.Rules) != len(b.Rules) {
		t.Fatalf("rules count: %d vs %d", len(a.Rules), len(b.Rules))
	}
	// Compare rules by name (unordered) since Write sorts by complexity
	bRules := make(map[string]grammar.Rule, len(b.Rules))
	for _, nr := range b.Rules {
		bRules[nr.Name] = nr.Rule
	}
	for _, nr := range a.Rules {
		br, ok := bRules[nr.Name]
		if !ok {
			t.Errorf("rule %q missing in round-tripped grammar", nr.Name)
			continue
		}
		assertRuleEqual(t, nr.Name, nr.Rule, br)
	}

	// Compare extras
	if len(a.Extras) != len(b.Extras) {
		t.Errorf("extras count: %d vs %d", len(a.Extras), len(b.Extras))
	} else {
		for i := range a.Extras {
			assertRuleEqual(t, "extras", a.Extras[i], b.Extras[i])
		}
	}

	// Compare other fields
	assertStringSliceEqual(t, "inline", a.Inline, b.Inline)
	assertStringSliceEqual(t, "supertypes", a.Supertypes, b.Supertypes)

	if len(a.Conflicts) != len(b.Conflicts) {
		t.Errorf("conflicts count: %d vs %d", len(a.Conflicts), len(b.Conflicts))
	}
}

func assertRuleEqual(t *testing.T, ctx string, a, b grammar.Rule) {
	t.Helper()

	if a.Type != b.Type {
		t.Errorf("%s: type %q vs %q", ctx, a.Type, b.Type)
		return
	}

	switch a.Type {
	case grammar.TypeSTRING:
		if a.StringValue() != b.StringValue() {
			t.Errorf("%s STRING: %q vs %q", ctx, a.StringValue(), b.StringValue())
		}
	case grammar.TypePATTERN:
		if a.StringValue() != b.StringValue() {
			t.Errorf("%s PATTERN: %q vs %q", ctx, a.StringValue(), b.StringValue())
		}
	case grammar.TypeSYMBOL:
		if a.Name != b.Name {
			t.Errorf("%s SYMBOL: %q vs %q", ctx, a.Name, b.Name)
		}
	case grammar.TypeSEQ, grammar.TypeCHOICE:
		if len(a.Members) != len(b.Members) {
			t.Errorf("%s %s members: %d vs %d", ctx, a.Type, len(a.Members), len(b.Members))
			return
		}
		for i := range a.Members {
			assertRuleEqual(t, fmt.Sprintf("%s/%s[%d]", ctx, a.Type, i), a.Members[i], b.Members[i])
		}
	case grammar.TypeREPEAT, grammar.TypeREPEAT1:
		assertRuleEqual(t, ctx+"/content", *a.Content, *b.Content)
	case grammar.TypePREC, grammar.TypePREC_LEFT, grammar.TypePREC_RIGHT, grammar.TypePREC_DYNAMIC:
		// Compare precedence values
		av, _ := a.IntValue()
		bv, _ := b.IntValue()
		if a.StringValue() != "" || b.StringValue() != "" {
			if a.StringValue() != b.StringValue() {
				t.Errorf("%s prec value: %q vs %q", ctx, a.StringValue(), b.StringValue())
			}
		} else if av != bv {
			t.Errorf("%s prec value: %d vs %d", ctx, av, bv)
		}
		assertRuleEqual(t, ctx+"/content", *a.Content, *b.Content)
	case grammar.TypeFIELD:
		if a.Name != b.Name {
			t.Errorf("%s FIELD name: %q vs %q", ctx, a.Name, b.Name)
		}
		assertRuleEqual(t, ctx+"/content", *a.Content, *b.Content)
	case grammar.TypeALIAS:
		if a.StringValue() != b.StringValue() {
			t.Errorf("%s ALIAS value: %q vs %q", ctx, a.StringValue(), b.StringValue())
		}
		aNamed := a.Named != nil && *a.Named
		bNamed := b.Named != nil && *b.Named
		if aNamed != bNamed {
			t.Errorf("%s ALIAS named: %v vs %v", ctx, aNamed, bNamed)
		}
		assertRuleEqual(t, ctx+"/content", *a.Content, *b.Content)
	case grammar.TypeTOKEN, grammar.TypeIMMEDIATE_TOKEN:
		assertRuleEqual(t, ctx+"/content", *a.Content, *b.Content)
	case grammar.TypeBLANK:
		// nothing to compare
	}
}

func assertStringSliceEqual(t *testing.T, name string, a, b []string) {
	t.Helper()
	if len(a) != len(b) {
		t.Errorf("%s count: %d vs %d", name, len(a), len(b))
		return
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("%s[%d]: %q vs %q", name, i, a[i], b[i])
		}
	}
}

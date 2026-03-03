package grammar

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseMinimal(t *testing.T) {
	data, err := os.ReadFile("../../testdata/json/minimal.json")
	if err != nil {
		t.Fatal(err)
	}

	var g Grammar
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatal(err)
	}

	if g.Name != "minimal" {
		t.Errorf("name = %q, want %q", g.Name, "minimal")
	}
	if len(g.Rules) != 4 {
		t.Fatalf("rules count = %d, want 4", len(g.Rules))
	}

	// Verify order preserved
	names := []string{"source", "_item", "word", "number"}
	for i, nr := range g.Rules {
		if nr.Name != names[i] {
			t.Errorf("rule[%d].Name = %q, want %q", i, nr.Name, names[i])
		}
	}

	// Verify source rule
	src := g.Rules[0].Rule
	if src.Type != TypeREPEAT {
		t.Errorf("source.Type = %q, want REPEAT", src.Type)
	}
	if src.Content == nil || src.Content.Type != TypeSYMBOL || src.Content.Name != "_item" {
		t.Errorf("source.Content unexpected: %+v", src.Content)
	}

	// Verify _item is CHOICE with 2 members
	item := g.Rules[1].Rule
	if item.Type != TypeCHOICE || len(item.Members) != 2 {
		t.Errorf("_item unexpected: type=%q members=%d", item.Type, len(item.Members))
	}

	// Verify number has BLANK in CHOICE (optional)
	num := g.Rules[3].Rule
	if num.Type != TypeSEQ || len(num.Members) != 2 {
		t.Fatalf("number unexpected: type=%q members=%d", num.Type, len(num.Members))
	}
	choice := num.Members[0]
	if choice.Type != TypeCHOICE || len(choice.Members) != 2 {
		t.Fatalf("number choice unexpected")
	}
	if choice.Members[1].Type != TypeBLANK {
		t.Errorf("expected BLANK, got %q", choice.Members[1].Type)
	}
}

func TestRoundTripJSON(t *testing.T) {
	data, err := os.ReadFile("../../testdata/json/minimal.json")
	if err != nil {
		t.Fatal(err)
	}

	var g Grammar
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatal(err)
	}

	out, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	// Re-parse and verify structural equality
	var g2 Grammar
	if err := json.Unmarshal(out, &g2); err != nil {
		t.Fatal(err)
	}

	if g.Name != g2.Name {
		t.Errorf("name mismatch: %q vs %q", g.Name, g2.Name)
	}
	if len(g.Rules) != len(g2.Rules) {
		t.Fatalf("rules count mismatch: %d vs %d", len(g.Rules), len(g2.Rules))
	}
	for i := range g.Rules {
		if g.Rules[i].Name != g2.Rules[i].Name {
			t.Errorf("rule[%d] name mismatch: %q vs %q", i, g.Rules[i].Name, g2.Rules[i].Name)
		}
	}
}

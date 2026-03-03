package abnf

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseMinimal(t *testing.T) {
	data, err := os.ReadFile("../../testdata/abnf/minimal.abnf")
	if err != nil {
		t.Fatal(err)
	}

	g, err := Parse(string(data))
	if err != nil {
		t.Fatal(err)
	}

	if g.Name != "minimal" {
		t.Errorf("name = %q, want %q", g.Name, "minimal")
	}
	if len(g.Rules) != 4 {
		t.Fatalf("rules count = %d, want 4", len(g.Rules))
	}

	names := []string{"source", "_item", "word", "number"}
	for i, nr := range g.Rules {
		if nr.Name != names[i] {
			t.Errorf("rule[%d].Name = %q, want %q", i, nr.Name, names[i])
		}
	}

	// source = *_item → REPEAT(SYMBOL(_item))
	src := g.Rules[0].Rule
	if src.Type != "REPEAT" {
		t.Errorf("source type = %q, want REPEAT", src.Type)
	}

	// _item = (word / number) → CHOICE
	item := g.Rules[1].Rule
	if item.Type != "CHOICE" {
		t.Errorf("_item type = %q, want CHOICE", item.Type)
	}

	// number = [%s"-"] @pattern("...") → SEQ(CHOICE(STRING, BLANK), PATTERN)
	num := g.Rules[3].Rule
	if num.Type != "SEQ" {
		t.Fatalf("number type = %q, want SEQ", num.Type)
	}
	if num.Members[0].Type != "CHOICE" {
		t.Errorf("number[0] type = %q, want CHOICE (optional)", num.Members[0].Type)
	}
}

func TestParseJSON(t *testing.T) {
	data, err := os.ReadFile("../../testdata/abnf/json.abnf")
	if err != nil {
		t.Fatal(err)
	}

	g, err := Parse(string(data))
	if err != nil {
		t.Fatal(err)
	}

	if g.Name != "json" {
		t.Errorf("name = %q, want %q", g.Name, "json")
	}

	// Basic structural checks
	if len(g.Rules) == 0 {
		t.Fatal("no rules parsed")
	}

	// Verify we can marshal back to JSON
	out, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Error("empty JSON output")
	}
}

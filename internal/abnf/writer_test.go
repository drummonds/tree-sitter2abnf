package abnf

import (
	"encoding/json"
	"flag"
	"os"
	"testing"

	"codeberg.org/hum3/tree-sitter2abnf/internal/grammar"
)

var update = flag.Bool("update", false, "update golden files")

func loadGrammar(t *testing.T, path string) *grammar.Grammar {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var g grammar.Grammar
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatal(err)
	}
	return &g
}

func TestWriteMinimal(t *testing.T) {
	g := loadGrammar(t, "../../testdata/json/minimal.json")
	got := Write(g)

	golden := "../../testdata/abnf/minimal.abnf"
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0644); err != nil {
			t.Fatal(err)
		}
		t.Log("updated golden file")
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden file missing (run with -update): %v", err)
	}
	if got != string(want) {
		t.Errorf("output mismatch.\ngot:\n%s\nwant:\n%s", got, string(want))
	}
}

func TestWriteJSON(t *testing.T) {
	g := loadGrammar(t, "../../testdata/json/json.json")
	got := Write(g)

	golden := "../../testdata/abnf/json.abnf"
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0644); err != nil {
			t.Fatal(err)
		}
		t.Log("updated golden file")
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden file missing (run with -update): %v", err)
	}
	if got != string(want) {
		t.Errorf("output mismatch.\ngot:\n%s\nwant:\n%s", got, string(want))
	}
}

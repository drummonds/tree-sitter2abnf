package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/drummonds/tree-sitter2abnf/internal/abnf"
	"github.com/drummonds/tree-sitter2abnf/internal/grammar"
)

var version = "dev"

func main() {
	output := flag.String("o", "", "output file (default: stdout)")
	flag.StringVar(output, "output", "", "output file (default: stdout)")
	showVersion := flag.Bool("v", false, "show version")
	flag.BoolVar(showVersion, "version", false, "show version")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: tree-sitter2abnf [flags] <input-file>\n\n")
		fmt.Fprintf(os.Stderr, "Converts between tree-sitter grammar.json and extended ABNF.\n")
		fmt.Fprintf(os.Stderr, "Direction auto-detected from file extension (.json → ABNF, .abnf → JSON).\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("tree-sitter2abnf", version)
		os.Exit(0)
	}

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	input := flag.Arg(0)
	data, err := os.ReadFile(input)
	if err != nil {
		fatal(err)
	}

	ext := strings.ToLower(filepath.Ext(input))
	var result []byte

	switch ext {
	case ".json":
		var g grammar.Grammar
		if err := json.Unmarshal(data, &g); err != nil {
			fatal(fmt.Errorf("parse %s: %w", input, err))
		}
		result = []byte(abnf.Write(&g))

	case ".abnf":
		g, err := abnf.Parse(string(data))
		if err != nil {
			fatal(fmt.Errorf("parse %s: %w", input, err))
		}
		result, err = json.MarshalIndent(g, "", "  ")
		if err != nil {
			fatal(err)
		}
		result = append(result, '\n')

	default:
		fatal(fmt.Errorf("unknown extension %q (expected .json or .abnf)", ext))
	}

	if *output != "" {
		if err := os.WriteFile(*output, result, 0644); err != nil {
			fatal(err)
		}
	} else {
		if _, err := os.Stdout.Write(result); err != nil {
			fatal(err)
		}
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

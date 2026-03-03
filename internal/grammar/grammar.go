// Package grammar provides Go types for tree-sitter grammar.json files.
package grammar

import (
	"encoding/json"
	"fmt"
	"io"
)

// Grammar represents a complete tree-sitter grammar.json file.
type Grammar struct {
	Name        string       `json:"name"`
	Word        string       `json:"word,omitempty"`
	Rules       []NamedRule  `json:"rules"`
	Extras      []Rule       `json:"extras,omitempty"`
	Conflicts   [][]string   `json:"conflicts,omitempty"`
	Precedences [][]PrecItem `json:"precedences,omitempty"`
	Externals   []Rule       `json:"externals,omitempty"`
	Inline      []string     `json:"inline,omitempty"`
	Supertypes  []string     `json:"supertypes,omitempty"`
}

// PrecItem is either a string (rule name) or a Rule (e.g. STRING literal).
type PrecItem struct {
	IsRule bool
	Name   string // if !IsRule
	Rule   Rule   // if IsRule
}

func (p *PrecItem) UnmarshalJSON(data []byte) error {
	// Try as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		p.Name = s
		return nil
	}
	// Otherwise it's a rule node
	p.IsRule = true
	return json.Unmarshal(data, &p.Rule)
}

func (p PrecItem) MarshalJSON() ([]byte, error) {
	if p.IsRule {
		return json.Marshal(p.Rule)
	}
	return json.Marshal(p.Name)
}

// NamedRule is a rule with its name, preserving key order from the JSON.
type NamedRule struct {
	Name string
	Rule Rule
}

// NodeType enumerates tree-sitter grammar node types.
type NodeType string

const (
	TypeSEQ             NodeType = "SEQ"
	TypeCHOICE          NodeType = "CHOICE"
	TypeREPEAT          NodeType = "REPEAT"
	TypeREPEAT1         NodeType = "REPEAT1"
	TypeSTRING          NodeType = "STRING"
	TypePATTERN         NodeType = "PATTERN"
	TypeSYMBOL          NodeType = "SYMBOL"
	TypeBLANK           NodeType = "BLANK"
	TypePREC            NodeType = "PREC"
	TypePREC_LEFT       NodeType = "PREC_LEFT"
	TypePREC_RIGHT      NodeType = "PREC_RIGHT"
	TypePREC_DYNAMIC    NodeType = "PREC_DYNAMIC"
	TypeFIELD           NodeType = "FIELD"
	TypeALIAS           NodeType = "ALIAS"
	TypeTOKEN           NodeType = "TOKEN"
	TypeIMMEDIATE_TOKEN NodeType = "IMMEDIATE_TOKEN"
)

// Rule is a polymorphic tree-sitter grammar node.
type Rule struct {
	Type NodeType `json:"type"`

	// SEQ, CHOICE: ordered children
	Members []Rule `json:"members,omitempty"`

	// REPEAT, REPEAT1, PREC, PREC_LEFT, PREC_RIGHT, PREC_DYNAMIC, TOKEN, IMMEDIATE_TOKEN, FIELD: single child
	Content *Rule `json:"content,omitempty"`

	// STRING: literal value
	// PATTERN: regex value
	// SYMBOL: rule name
	Value any `json:"value,omitempty"`

	// SYMBOL
	Name string `json:"name,omitempty"`

	// PREC, PREC_LEFT, PREC_RIGHT, PREC_DYNAMIC: precedence (int or string)
	// (stored in Value)

	// FIELD: field name (stored in Name), content in Content

	// ALIAS
	Named *bool `json:"named,omitempty"`
	// ALIAS value is in Value, content in Content
}

// StringValue returns Value as a string, or empty string.
func (r Rule) StringValue() string {
	switch v := r.Value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

// IntValue returns Value as an int. Returns 0 and false if not numeric.
func (r Rule) IntValue() (int, bool) {
	switch v := r.Value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	default:
		return 0, false
	}
}

// PrecValue returns the precedence as either int or string.
func (r Rule) PrecValue() any {
	return r.Value
}

// custom JSON handling for Grammar to preserve rule order

type grammarJSON struct {
	Name        string          `json:"name"`
	Word        string          `json:"word,omitempty"`
	Rules       json.RawMessage `json:"rules"`
	Extras      []Rule          `json:"extras,omitempty"`
	Conflicts   [][]string      `json:"conflicts,omitempty"`
	Precedences [][]PrecItem    `json:"precedences,omitempty"`
	Externals   []Rule          `json:"externals,omitempty"`
	Inline      []string        `json:"inline,omitempty"`
	Supertypes  []string        `json:"supertypes,omitempty"`
}

// UnmarshalJSON implements custom unmarshalling that preserves rule order.
func (g *Grammar) UnmarshalJSON(data []byte) error {
	var raw grammarJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal grammar: %w", err)
	}

	g.Name = raw.Name
	g.Word = raw.Word
	g.Extras = raw.Extras
	g.Conflicts = raw.Conflicts
	g.Precedences = raw.Precedences
	g.Externals = raw.Externals
	g.Inline = raw.Inline
	g.Supertypes = raw.Supertypes

	// Parse rules preserving key order
	rules, err := parseOrderedRules(raw.Rules)
	if err != nil {
		return fmt.Errorf("unmarshal rules: %w", err)
	}
	g.Rules = rules
	return nil
}

// parseOrderedRules parses a JSON object into ordered NamedRules.
// Uses json.Decoder to preserve key order.
func parseOrderedRules(data json.RawMessage) ([]NamedRule, error) {
	// We need to parse the object preserving key order.
	// Go's encoding/json doesn't guarantee map order, but json.Decoder
	// with Token() reads keys in order.
	dec := json.NewDecoder(bytesReader(data))

	// Read opening {
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := t.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("expected {, got %v", t)
	}

	var rules []NamedRule
	for dec.More() {
		// Read key
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := t.(string)
		if !ok {
			return nil, fmt.Errorf("expected string key, got %v", t)
		}

		// Read value
		var rule Rule
		if err := dec.Decode(&rule); err != nil {
			return nil, fmt.Errorf("rule %q: %w", key, err)
		}
		rules = append(rules, NamedRule{Name: key, Rule: rule})
	}

	return rules, nil
}

// MarshalJSON implements custom marshalling that preserves rule order.
func (g Grammar) MarshalJSON() ([]byte, error) {
	// Build rules as ordered JSON
	rulesJSON, err := marshalOrderedRules(g.Rules)
	if err != nil {
		return nil, err
	}

	raw := grammarJSON{
		Name:        g.Name,
		Word:        g.Word,
		Rules:       rulesJSON,
		Extras:      g.Extras,
		Conflicts:   g.Conflicts,
		Precedences: g.Precedences,
		Externals:   g.Externals,
		Inline:      g.Inline,
		Supertypes:  g.Supertypes,
	}
	return json.Marshal(raw)
}

func marshalOrderedRules(rules []NamedRule) (json.RawMessage, error) {
	buf := []byte{'{'}
	for i, nr := range rules {
		if i > 0 {
			buf = append(buf, ',')
		}
		key, err := json.Marshal(nr.Name)
		if err != nil {
			return nil, err
		}
		val, err := json.Marshal(nr.Rule)
		if err != nil {
			return nil, err
		}
		buf = append(buf, key...)
		buf = append(buf, ':')
		buf = append(buf, val...)
	}
	buf = append(buf, '}')
	return json.RawMessage(buf), nil
}

type bytesReaderImpl struct {
	data []byte
	pos  int
}

func bytesReader(data []byte) *bytesReaderImpl {
	return &bytesReaderImpl{data: data}
}

func (r *bytesReaderImpl) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

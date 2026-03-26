package abnf

import (
	"fmt"
	"codeberg.org/hum3/tree-sitter2abnf/internal/grammar"
	"strconv"
	"strings"
)

// Parse converts extended ABNF text back into a Grammar.
func Parse(input string) (*grammar.Grammar, error) {
	p := &parser{
		input: input,
		pos:   0,
		line:  1,
		col:   1,
	}
	return p.parseGrammar()
}

type parser struct {
	input string
	pos   int
	line  int
	col   int
}

func (p *parser) errorf(format string, args ...any) error {
	return fmt.Errorf("line %d col %d: "+format, append([]any{p.line, p.col}, args...)...)
}

func (p *parser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

func (p *parser) advance() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	ch := p.input[p.pos]
	p.pos++
	if ch == '\n' {
		p.line++
		p.col = 1
	} else {
		p.col++
	}
	return ch
}

func (p *parser) eof() bool {
	return p.pos >= len(p.input)
}

func (p *parser) skipSpaces() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t') {
		p.advance()
	}
}

func (p *parser) skipWhitespace() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t' || p.input[p.pos] == '\r' || p.input[p.pos] == '\n') {
		p.advance()
	}
}

func (p *parser) expect(ch byte) error {
	if p.eof() || p.input[p.pos] != ch {
		return p.errorf("expected %q, got %q", string(ch), string(p.peek()))
	}
	p.advance()
	return nil
}

func (p *parser) parseGrammar() (*grammar.Grammar, error) {
	g := &grammar.Grammar{}

	// Parse directives and rules
	for !p.eof() {
		p.skipWhitespace()
		if p.eof() {
			break
		}

		if p.peek() == ';' {
			if err := p.parseDirective(g); err != nil {
				return nil, err
			}
			continue
		}

		// Must be a rule definition
		nr, err := p.parseRuleDef()
		if err != nil {
			return nil, err
		}
		g.Rules = append(g.Rules, nr)
	}

	return g, nil
}

func (p *parser) parseDirective(g *grammar.Grammar) error {
	p.advance() // skip ';'
	p.skipSpaces()

	// Read directive name
	if p.eof() || p.peek() == '\n' {
		p.skipToEOL()
		return nil
	}

	if p.peek() != '@' {
		// Regular comment, skip
		p.skipToEOL()
		return nil
	}

	name := p.readWord()
	p.skipSpaces()

	switch name {
	case DirGrammar:
		s, err := p.readQuotedString()
		if err != nil {
			return err
		}
		g.Name = s

	case DirWord:
		s, err := p.readQuotedString()
		if err != nil {
			return err
		}
		g.Word = ToGrammarName(s)

	case DirExtras:
		rules, err := p.parseDirectiveRuleList()
		if err != nil {
			return err
		}
		g.Extras = rules

	case DirInline:
		names, err := p.parseNameList()
		if err != nil {
			return err
		}
		g.Inline = names

	case DirConflicts:
		for {
			p.skipSpaces()
			if p.eof() || p.peek() == '\n' {
				break
			}
			if p.peek() != '(' {
				break
			}
			group, err := p.parseNameGroup()
			if err != nil {
				return err
			}
			g.Conflicts = append(g.Conflicts, group)
		}

	case DirExternals:
		rules, err := p.parseDirectiveRuleList()
		if err != nil {
			return err
		}
		g.Externals = rules

	case DirSupertypes:
		names, err := p.parseNameList()
		if err != nil {
			return err
		}
		g.Supertypes = names

	default:
		// Unknown directive, skip
	}

	p.skipToEOL()
	return nil
}

func (p *parser) parseDirectiveRuleList() ([]grammar.Rule, error) {
	if err := p.expect('('); err != nil {
		return nil, err
	}
	var rules []grammar.Rule
	for {
		p.skipSpaces()
		if p.peek() == ')' {
			p.advance()
			break
		}
		if p.peek() == '/' {
			p.advance()
			p.skipSpaces()
			continue
		}
		r, err := p.parseAnnotatedAtom()
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, nil
}

func (p *parser) parseNameList() ([]string, error) {
	if err := p.expect('('); err != nil {
		return nil, err
	}
	var names []string
	for {
		p.skipSpaces()
		if p.peek() == ')' {
			p.advance()
			break
		}
		if p.peek() == '/' {
			p.advance()
			p.skipSpaces()
			continue
		}
		name := p.readWord()
		if name == "" {
			return nil, p.errorf("expected name")
		}
		names = append(names, ToGrammarName(name))
	}
	return names, nil
}

func (p *parser) parseNameGroup() ([]string, error) {
	if err := p.expect('('); err != nil {
		return nil, err
	}
	var names []string
	for {
		p.skipSpaces()
		if p.peek() == ')' {
			p.advance()
			break
		}
		name := p.readWord()
		if name == "" {
			return nil, p.errorf("expected name in conflict group")
		}
		names = append(names, ToGrammarName(name))
	}
	return names, nil
}

func (p *parser) parseRuleDef() (grammar.NamedRule, error) {
	name := p.readWord()
	if name == "" {
		return grammar.NamedRule{}, p.errorf("expected rule name")
	}
	name = ToGrammarName(name)
	p.skipSpaces()

	// Expect '=' or '=/'
	if err := p.expect('='); err != nil {
		return grammar.NamedRule{}, err
	}
	if p.peek() == '/' {
		p.advance() // incremental alternative, we handle as main
	}
	p.skipSpaces()

	// If the rule body starts on the next line (long-form), handle that
	if p.peek() == '\n' || p.peek() == '\r' {
		p.skipWhitespace()
	}

	rule, err := p.parseAlternation()
	if err != nil {
		return grammar.NamedRule{}, fmt.Errorf("rule %q: %w", name, err)
	}

	return grammar.NamedRule{Name: name, Rule: rule}, nil
}

// parseAlternation parses: expr ("/" expr)*
// Handles multi-line alternatives where continuation lines start with whitespace.
func (p *parser) parseAlternation() (grammar.Rule, error) {
	first, err := p.parseConcatenation()
	if err != nil {
		return grammar.Rule{}, err
	}

	var members []grammar.Rule
	for {
		p.skipSpaces()
		// Check for "/" alternation
		if p.peek() == '/' {
			p.advance()
			p.skipSpaces()
			// Handle newline after /
			if p.peek() == '\n' || p.peek() == '\r' {
				p.skipWhitespace()
			}
		} else if p.peek() == '\n' || p.peek() == '\r' {
			// Check if next line is a continuation (starts with whitespace then /)
			saved := p.savePos()
			p.skipWhitespace()
			if p.peek() == '/' {
				p.advance()
				p.skipSpaces()
			} else {
				p.restorePos(saved)
				break
			}
		} else {
			break
		}

		next, err := p.parseConcatenation()
		if err != nil {
			return grammar.Rule{}, err
		}
		if members == nil {
			members = append(members, first)
		}
		members = append(members, next)
	}

	if members != nil {
		return grammar.Rule{Type: grammar.TypeCHOICE, Members: members}, nil
	}
	return first, nil
}

// parseConcatenation parses: repetition (SP repetition)*
func (p *parser) parseConcatenation() (grammar.Rule, error) {
	first, err := p.parseAnnotatedAtom()
	if err != nil {
		return grammar.Rule{}, err
	}

	var members []grammar.Rule
	for {
		p.skipSpaces()
		if p.eof() || p.peek() == '/' || p.peek() == ')' || p.peek() == ']' || p.peek() == '\n' || p.peek() == '\r' {
			break
		}
		next, err := p.parseAnnotatedAtom()
		if err != nil {
			return grammar.Rule{}, err
		}
		if members == nil {
			members = append(members, first)
		}
		members = append(members, next)
	}

	if members != nil {
		return grammar.Rule{Type: grammar.TypeSEQ, Members: members}, nil
	}
	return first, nil
}

// parseAnnotatedAtom handles @annotation prefixed atoms.
func (p *parser) parseAnnotatedAtom() (grammar.Rule, error) {
	if p.peek() == '@' {
		return p.parseAnnotation()
	}
	return p.parseRepetition()
}

func (p *parser) parseAnnotation() (grammar.Rule, error) {
	name := p.readWord() // reads @xxx
	switch name {
	case AnnPrec:
		return p.parsePrecAnnotation(grammar.TypePREC)
	case AnnPrecLeft:
		return p.parsePrecAnnotation(grammar.TypePREC_LEFT)
	case AnnPrecRight:
		return p.parsePrecAnnotation(grammar.TypePREC_RIGHT)
	case AnnPrecDynamic:
		return p.parsePrecAnnotation(grammar.TypePREC_DYNAMIC)

	case AnnField:
		if err := p.expect('('); err != nil {
			return grammar.Rule{}, err
		}
		fieldName := p.readWord()
		if err := p.expect(')'); err != nil {
			return grammar.Rule{}, err
		}
		p.skipSpaces()
		content, err := p.parseAnnotatedAtom()
		if err != nil {
			return grammar.Rule{}, err
		}
		return grammar.Rule{
			Type:    grammar.TypeFIELD,
			Name:    fieldName,
			Content: &content,
		}, nil

	case AnnAlias:
		if err := p.expect('('); err != nil {
			return grammar.Rule{}, err
		}
		named := false
		if p.peek() == '~' {
			p.advance()
			named = true
		}
		aliasName := p.readUntil(')')
		if err := p.expect(')'); err != nil {
			return grammar.Rule{}, err
		}
		p.skipSpaces()
		content, err := p.parseAnnotatedAtom()
		if err != nil {
			return grammar.Rule{}, err
		}
		r := grammar.Rule{
			Type:    grammar.TypeALIAS,
			Content: &content,
			Value:   aliasName,
		}
		if named {
			r.Named = boolPtr(true)
		} else {
			r.Named = boolPtr(false)
		}
		return r, nil

	case AnnToken:
		return p.parseTokenAnnotation(grammar.TypeTOKEN)

	case AnnImmToken:
		return p.parseTokenAnnotation(grammar.TypeIMMEDIATE_TOKEN)

	case AnnPattern:
		if err := p.expect('('); err != nil {
			return grammar.Rule{}, err
		}
		s, err := p.readQuotedString()
		if err != nil {
			return grammar.Rule{}, err
		}
		if err := p.expect(')'); err != nil {
			return grammar.Rule{}, err
		}
		return grammar.Rule{Type: grammar.TypePATTERN, Value: s}, nil

	default:
		return grammar.Rule{}, p.errorf("unknown annotation %q", name)
	}
}

func (p *parser) parsePrecAnnotation(typ grammar.NodeType) (grammar.Rule, error) {
	if err := p.expect('('); err != nil {
		return grammar.Rule{}, err
	}
	val, err := p.parsePrecValue()
	if err != nil {
		return grammar.Rule{}, err
	}
	if err := p.expect(')'); err != nil {
		return grammar.Rule{}, err
	}
	p.skipSpaces()
	content, err := p.parseAnnotatedAtom()
	if err != nil {
		return grammar.Rule{}, err
	}
	return grammar.Rule{
		Type:    typ,
		Value:   val,
		Content: &content,
	}, nil
}

func (p *parser) parsePrecValue() (any, error) {
	// Could be an integer (possibly negative) or a string name
	start := p.pos
	if p.peek() == '-' || (p.peek() >= '0' && p.peek() <= '9') {
		if p.peek() == '-' {
			p.advance()
		}
		for p.pos < len(p.input) && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
			p.advance()
		}
		n, err := strconv.Atoi(p.input[start:p.pos])
		if err != nil {
			return nil, p.errorf("invalid precedence number: %w", err)
		}
		return float64(n), nil // JSON numbers are float64
	}
	// String precedence name
	name := p.readUntil(')')
	if name == "" {
		return nil, p.errorf("expected precedence value")
	}
	return name, nil
}

func (p *parser) parseTokenAnnotation(typ grammar.NodeType) (grammar.Rule, error) {
	if err := p.expect('('); err != nil {
		return grammar.Rule{}, err
	}
	// Parse the inner content as an alternation (it may contain / separators)
	content, err := p.parseAlternation()
	if err != nil {
		return grammar.Rule{}, err
	}
	if err := p.expect(')'); err != nil {
		return grammar.Rule{}, err
	}
	return grammar.Rule{
		Type:    typ,
		Content: &content,
	}, nil
}

// parseRepetition handles: [N]*expr or *expr or 1*expr
func (p *parser) parseRepetition() (grammar.Rule, error) {
	if p.peek() == '*' {
		p.advance()
		inner, err := p.parseAtom()
		if err != nil {
			return grammar.Rule{}, err
		}
		return grammar.Rule{Type: grammar.TypeREPEAT, Content: &inner}, nil
	}
	if p.peek() == '1' && p.pos+1 < len(p.input) && p.input[p.pos+1] == '*' {
		p.advance() // '1'
		p.advance() // '*'
		inner, err := p.parseAtom()
		if err != nil {
			return grammar.Rule{}, err
		}
		return grammar.Rule{Type: grammar.TypeREPEAT1, Content: &inner}, nil
	}
	return p.parseAtom()
}

// parseAtom handles: group, optional, string, rule name
func (p *parser) parseAtom() (grammar.Rule, error) {
	switch p.peek() {
	case '(':
		p.advance()
		p.skipSpaces()
		inner, err := p.parseAlternation()
		if err != nil {
			return grammar.Rule{}, err
		}
		p.skipSpaces()
		if err := p.expect(')'); err != nil {
			return grammar.Rule{}, err
		}
		return inner, nil

	case '[':
		p.advance()
		p.skipSpaces()
		inner, err := p.parseAlternation()
		if err != nil {
			return grammar.Rule{}, err
		}
		p.skipSpaces()
		if err := p.expect(']'); err != nil {
			return grammar.Rule{}, err
		}
		// Optional = CHOICE(inner, BLANK)
		return grammar.Rule{
			Type: grammar.TypeCHOICE,
			Members: []grammar.Rule{
				inner,
				{Type: grammar.TypeBLANK},
			},
		}, nil

	case '%':
		// %s"..." case-sensitive string
		p.advance() // %
		if p.peek() == 's' {
			p.advance() // s
		}
		s, err := p.readQuotedString()
		if err != nil {
			return grammar.Rule{}, err
		}
		return grammar.Rule{Type: grammar.TypeSTRING, Value: s}, nil

	case '"':
		// Bare quoted string (shouldn't normally appear, but handle it)
		s, err := p.readQuotedString()
		if err != nil {
			return grammar.Rule{}, err
		}
		return grammar.Rule{Type: grammar.TypeSTRING, Value: s}, nil

	default:
		// Must be a rule name (ALPHA / "_") *(ALPHA / DIGIT / "-" / "_")
		if isNameStart(p.peek()) {
			name := ToGrammarName(p.readWord())
			return grammar.Rule{Type: grammar.TypeSYMBOL, Name: name}, nil
		}
		return grammar.Rule{}, p.errorf("unexpected character %q", string(p.peek()))
	}
}

// Helper methods

func (p *parser) readWord() string {
	start := p.pos
	for p.pos < len(p.input) && isNameChar(p.input[p.pos]) {
		p.advance()
	}
	return p.input[start:p.pos]
}

func (p *parser) readUntil(ch byte) string {
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] != ch {
		p.advance()
	}
	return p.input[start:p.pos]
}

func (p *parser) readQuotedString() (string, error) {
	if p.peek() != '"' {
		return "", p.errorf("expected '\"', got %q", string(p.peek()))
	}
	p.advance()
	var sb strings.Builder
	for {
		if p.eof() {
			return "", p.errorf("unterminated string")
		}
		ch := p.peek()
		if ch == '"' {
			p.advance()
			return sb.String(), nil
		}
		if ch == '\\' {
			p.advance()
			esc := p.advance()
			switch esc {
			case '"':
				sb.WriteByte('"')
			case '\\':
				sb.WriteByte('\\')
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			default:
				sb.WriteByte('\\')
				sb.WriteByte(esc)
			}
		} else {
			sb.WriteByte(ch)
			p.advance()
		}
	}
}

func (p *parser) skipToEOL() {
	for p.pos < len(p.input) && p.input[p.pos] != '\n' {
		p.advance()
	}
	if p.pos < len(p.input) {
		p.advance() // skip \n
	}
}

type savedPos struct {
	pos  int
	line int
	col  int
}

func (p *parser) savePos() savedPos {
	return savedPos{pos: p.pos, line: p.line, col: p.col}
}

func (p *parser) restorePos(s savedPos) {
	p.pos = s.pos
	p.line = s.line
	p.col = s.col
}

func isNameStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' || ch == '@'
}

func isNameChar(ch byte) bool {
	return isNameStart(ch) || (ch >= '0' && ch <= '9') || ch == '-'
}

func boolPtr(b bool) *bool {
	return &b
}

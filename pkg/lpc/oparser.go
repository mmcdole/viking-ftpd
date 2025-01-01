package lpc

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ObjectParser holds parsing configuration
type ObjectParser struct {
	strict bool
}

// NewObjectParser creates a new parser with the given options
func NewObjectParser(strict bool) *ObjectParser {
	return &ObjectParser{
		strict: strict,
	}
}

// ParseError represents an error that occurred while parsing a specific line
type ParseError struct {
	Line     string
	Position int
	Err      error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("error at position %d: %v", e.Position, e.Err)
}

// ParseResult contains both the parsed object and any errors encountered
type ParseResult struct {
	Object map[string]interface{}
	Errors []*ParseError
}

// LineParser handles parsing of individual lines
type LineParser struct {
	s   string
	pos int
	w   int
}

// NewLineParser creates a new parser for a single line
func NewLineParser(line string) *LineParser {
	return &LineParser{
		s:   line,
		pos: 0,
		w:   len(line),
	}
}

// ParseObject parses an LPC object from the input string
func (p *ObjectParser) ParseObject(input string) (*ParseResult, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("input string is empty")
	}

	result := &ParseResult{
		Object: make(map[string]interface{}),
		Errors: make([]*ParseError, 0),
	}

	lines := strings.Split(input, "\n")
	startPos := 0

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			startPos += len(line) + 1 // +1 for newline
			continue
		}

		lp := NewLineParser(line)
		key, value, err := lp.ParseLine()
		if err != nil {
			parseErr := &ParseError{
				Line:     line,
				Position: startPos + lp.pos,
				Err:      err,
			}
			if p.strict {
				return nil, parseErr
			}
			result.Errors = append(result.Errors, parseErr)
		} else if key != "" {
			result.Object[key] = value
		}

		startPos += len(line) + 1 // +1 for newline
	}

	return result, nil
}

// ParseLine parses a single line of LPC object format, returning the key and value.
// For blueprint lines, the key will be "blueprint" and the value will be the blueprint string.
func (p *LineParser) ParseLine() (string, interface{}, error) {
	if p.peek(0) == '#' {
		blueprint, err := p.parseBlueprint()
		if err != nil {
			return "", nil, err
		}
		return "blueprint", blueprint, nil
	}

	key, err := p.parseIdentifier()
	if err != nil {
		return "", nil, err
	}
	p.skipSpaces()
	value, err := p.parseValue()
	if err != nil {
		return "", nil, err
	}

	r := p.peek(0)
	if r != '\n' && r != 0 {
		return "", nil, fmt.Errorf("expected new line or end of file at position %d", p.pos)
	}

	return key, value, nil
}

func (p *LineParser) parseValue() (interface{}, error) {
	p.skipSpaces()
	r := p.peek(0)

	if r == '"' {
		return p.parseString()
	} else if unicode.IsDigit(r) || r == '-' {
		return p.parseNumber()
	} else if r == '(' {
		if p.peek(1) == '[' {
			return p.parseMap()
		} else if p.peek(1) == '{' {
			return p.parseList()
		}
	}
	return nil, fmt.Errorf("invalid value type '%c' at position %d", r, p.pos)
}

func (p *LineParser) parseNumber() (interface{}, error) {
	// Look ahead to see if this is a float
	isFloat := false
	offset := 0
	if p.peek(offset) == '-' {
		offset++
	}
	for unicode.IsDigit(p.peek(offset)) || p.peek(offset) == '.' {
		if p.peek(offset) == '.' {
			isFloat = true
		}
		offset++
	}
	if p.peek(offset) == '=' {
		isFloat = true
	}
	if isFloat {
		return p.parseFloat()
	}
	return p.parseInt()
}

func (p *LineParser) parseFloat() (float64, error) {
	start := p.pos

	if p.peek(0) == '-' {
		p.next()
	}

	for unicode.IsDigit(p.peek(0)) || p.peek(0) == '.' {
		p.next()
	}

	floatStr := p.s[start:p.pos]
	result, err := strconv.ParseFloat(floatStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float value %q at position %d: %w", floatStr, p.pos, err)
	}

	if p.peek(0) == '=' {
		p.next() // skip =
		for {
			r := p.peek(0)
			if r == 0 {
				break
			}
			isDigit := r >= '0' && r <= '9'
			isLowerHex := r >= 'a' && r <= 'f'
			isUpperHex := r >= 'A' && r <= 'F'
			if !(isDigit || isLowerHex || isUpperHex) {
				break
			}
			p.next()
		}
	}

	return result, nil
}

func (p *LineParser) parseIdentifier() (string, error) {
	start := p.pos
	for r := p.next(); unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'; r = p.next() {
		if r == 0 {
			return "", fmt.Errorf("unexpected end of input while parsing identifier at position %d", p.pos)
		}
	}
	result := p.s[start : p.pos-p.w]
	p.skipSpaces()
	return result, nil
}

func (p *LineParser) parseMap() (map[string]interface{}, error) {
	if !p.expect('(') || !p.expect('[') {
		return nil, fmt.Errorf("error in map: expected '([' at position %d", p.pos)
	}
	p.skipSpaces()
	size, err := p.parseInt()
	if err != nil {
		return nil, fmt.Errorf("error in map: invalid size at position %d", p.pos)
	}
	if !p.expect('|') {
		return nil, fmt.Errorf("error in map: expected '|' at position %d", p.pos)
	}

	entries := make(map[string]interface{})
	for i := 0; i < size; i++ {
		key, value, err := p.parseMapEntry()
		if err != nil {
			return nil, fmt.Errorf("error in map entry at position %d: %w", p.pos, err)
		}
		entries[key] = value
		p.expect(',') // Allow trailing comma
	}

	if !p.expect(']') || !p.expect(')') {
		return nil, fmt.Errorf("error in map: expected '])' at position %d", p.pos)
	}

	p.skipSpaces()
	return entries, nil
}

func (p *LineParser) parseMapEntry() (string, interface{}, error) {
	keyValue, err := p.parseValue()
	if err != nil {
		return "", nil, fmt.Errorf("error in map entry: invalid key at position %d: %v", p.pos, err)
	}

	var key string
	switch k := keyValue.(type) {
	case string:
		key = k
	case int:
		key = strconv.Itoa(k)
	case []interface{}:
		parts := make([]string, len(k))
		for i, item := range k {
			switch iv := item.(type) {
			case string:
				parts[i] = iv
			case int:
				parts[i] = strconv.Itoa(iv)
			default:
				return "", nil, fmt.Errorf("error in map entry: list key contains unsupported type at position %d (got %T)", p.pos, item)
			}
		}
		key = strings.Join(parts, ",")
	default:
		return "", nil, fmt.Errorf("error in map entry: key must be string, integer, or list at position %d (got %T)", p.pos, keyValue)
	}

	p.skipSpaces()
	r := p.peek(0)
	if !p.expect(':') {
		return "", nil, fmt.Errorf("error in map entry: expected ':' at position %d, got '%c'", p.pos, r)
	}
	p.skipSpaces()
	value, err := p.parseValue()
	if err != nil {
		return "", nil, fmt.Errorf("error in map entry: invalid value at position %d: %v", p.pos, err)
	}
	p.skipSpaces()
	return key, value, nil
}

func (p *LineParser) parseList() ([]interface{}, error) {
	if !p.expect('(') || !p.expect('{') {
		return nil, fmt.Errorf("error in list: expected '({' at position %d", p.pos)
	}
	p.skipSpaces()
	size, err := p.parseInt()
	if err != nil {
		return nil, fmt.Errorf("error in list: invalid size at position %d", p.pos)
	}
	if !p.expect('|') {
		return nil, fmt.Errorf("error in list: expected '|' at position %d", p.pos)
	}

	list := make([]interface{}, 0, size)
	for i := 0; i < size; i++ {
		value, err := p.parseValue()
		if err != nil {
			return nil, fmt.Errorf("error in list: invalid value at position %d", p.pos)
		}
		list = append(list, value)
		p.expect(',') // Allow trailing comma
	}

	if !p.expect('}') || !p.expect(')') {
		return nil, fmt.Errorf("error in list: expected '})' at position %d", p.pos)
	}
	return list, nil
}

func (p *LineParser) parseString() (string, error) {
	if !p.expect('"') {
		return "", fmt.Errorf("error in string: expected '\"' at position %d", p.pos)
	}
	var result strings.Builder
	for {
		r := p.peek(0)
		if r == 0 {
			return "", fmt.Errorf("error in string: unterminated string at position %d", p.pos)
		}
		if r == '\\' {
			p.next()
			r = p.peek(0)
			if r == '"' {
				result.WriteRune('"')
			} else {
				result.WriteRune('\\')
				result.WriteRune(r)
			}
			p.next()
			continue
		}
		if r == '"' {
			break
		}
		result.WriteRune(r)
		p.next()
	}
	if !p.expect('"') {
		return "", fmt.Errorf("error in string: expected '\"' at position %d", p.pos)
	}
	return result.String(), nil
}

func (p *LineParser) parseInt() (int, error) {
	start := p.pos
	if p.peek(0) == '-' {
		p.next()
	}
	for unicode.IsDigit(p.peek(0)) {
		p.next()
	}

	result, err := strconv.Atoi(p.s[start:p.pos])
	if err != nil {
		return 0, fmt.Errorf("error in integer: invalid number at position %d", p.pos)
	}
	return result, nil
}

func (p *LineParser) parseBlueprint() (string, error) {
	if p.peek(0) != '#' {
		return "", fmt.Errorf("expected blueprint to start with #")
	}
	p.next() // skip #

	var blueprint strings.Builder
	for p.peek(0) != 0 && p.peek(0) != '\n' {
		blueprint.WriteRune(p.next())
	}
	if p.peek(0) == '\n' {
		p.next()
	}
	return blueprint.String(), nil
}

func (p *LineParser) peek(n int) rune {
	if p.pos+n >= len(p.s) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(p.s[p.pos+n:])
	return r
}

func (p *LineParser) next() rune {
	if p.pos >= len(p.s) {
		return 0
	}
	r, w := utf8.DecodeRuneInString(p.s[p.pos:])
	p.pos += w
	p.w = w
	return r
}

func (p *LineParser) expect(r rune) bool {
	if p.next() == r {
		return true
	}
	p.pos -= p.w
	return false
}

func (p *LineParser) skipSpaces() {
	for {
		r := p.peek(0)
		if r != ' ' && r != '\t' {
			break
		}
		p.next()
	}
}

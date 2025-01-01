package lpc

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type ObjectParser struct {
	s   string
	pos int
	w   int
}

func NewObjectParser(input string) *ObjectParser {
	return &ObjectParser{
		s:   input,
		pos: 0,
		w:   len(input),
	}
}

// ParseObject parses an entire LPC object from the input string.
func (p *ObjectParser) ParseObject() (map[string]interface{}, error) {
	if p.w == 0 {
		return nil, fmt.Errorf("input string is empty")
	}

	object := make(map[string]interface{})
	for p.peek(0) != 0 {
		key, value, err := p.parseLine()
		if err != nil {
			return nil, fmt.Errorf("error parsing object line at position %d: %w", p.pos, err)
		}
		object[key] = value
	}
	return object, nil
}

func (p *ObjectParser) parseLine() (string, interface{}, error) {
	key, err := p.parseIdentifier()
	if err != nil {
		return "", nil, err
	}
	p.skipSpaces()
	value, err := p.parseValue()
	if err != nil {
		return "", nil, err
	}

	// expect a new line or end of file
	r := p.peek(0)
	if r != '\n' && r != 0 {
		return "", nil, fmt.Errorf("expected new line or end of file at position %d", p.pos)
	}
	if r == '\n' {
		p.next() // consume the new line
	}
	return key, value, nil
}

func (p *ObjectParser) parseValue() (interface{}, error) {
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

func (p *ObjectParser) parseNumber() (interface{}, error) {
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
	// Check if there's an '=' indicating the new float type
	if p.peek(offset) == '=' {
		isFloat = true
	}
	if isFloat {
		return p.parseFloat()
	}
	return p.parseInt()
}

func (p *ObjectParser) parseFloat() (float64, error) {
	start := p.pos

	// Skip optional negative sign
	if p.peek(0) == '-' {
		p.next()
	}

	// Consume digits and decimal point
	for unicode.IsDigit(p.peek(0)) || p.peek(0) == '.' {
		p.next()
	}

	// Parse the float part
	floatStr := p.s[start:p.pos]
	result, err := strconv.ParseFloat(floatStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float value %q at position %d: %w", floatStr, p.pos, err)
	}

	// If there's an '=', consume the hex part
	if p.peek(0) == '=' {
		p.next() // skip =

		// Consume all hex digits (0-9, a-f, A-F)
		for {
			r := p.peek(0)
			if r == 0 { // end of input
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

func (p *ObjectParser) parseIdentifier() (string, error) {
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

func (p *ObjectParser) parseMap() (map[string]interface{}, error) {
	if !p.expect('(') || !p.expect('[') {
		return nil, fmt.Errorf("error in map: expected '([' at position %d", p.pos)
	}
	p.skipSpaces()
	entryCount, err := p.parseInt()
	if err != nil {
		return nil, fmt.Errorf("error in map: invalid entry count at position %d", p.pos)
	}
	if !p.expect('|') {
		return nil, fmt.Errorf("error in map: expected '|' delimiter at position %d", p.pos)
	}
	p.skipSpaces()

	entries, err := p.parseMapEntryList(entryCount)
	if err != nil {
		return nil, fmt.Errorf("error in map: %w", err)
	}

	if !p.expect(']') || !p.expect(')') {
		return nil, fmt.Errorf("error in map: expected '])' at position %d", p.pos)
	}

	p.skipSpaces()

	return entries, nil
}

func (p *ObjectParser) parseMapEntryList(entryCount int) (map[string]interface{}, error) {
	entries := make(map[string]interface{})
	for i := 0; i < entryCount; i++ {
		key, value, err := p.parseMapEntry()
		if err != nil {
			return nil, fmt.Errorf("error in map entry at position %d: %w", p.pos, err)
		}
		entries[key] = value
		p.expect(',') // Allow trailing comma
	}
	return entries, nil
}

func (p *ObjectParser) parseMapEntry() (string, interface{}, error) {
	keyValue, err := p.parseValue()
	if err != nil {
		return "", nil, fmt.Errorf("error in map entry: invalid key at position %d: %v", p.pos, err)
	}

	var key string
	switch v := keyValue.(type) {
	case string:
		key = v
	case int:
		key = strconv.Itoa(v)
	default:
		return "", nil, fmt.Errorf("error in map entry: key must be string or integer at position %d (got %T)", p.pos, keyValue)
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

func (p *ObjectParser) parseList() ([]interface{}, error) {
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

	if !p.expect('}') {
		return nil, fmt.Errorf("error in list: expected '}' at position %d", p.pos)
	}
	if !p.expect(')') {
		return nil, fmt.Errorf("error in list: expected ')' at position %d", p.pos)
	}
	return list, nil
}

func (p *ObjectParser) parseString() (string, error) {
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
			// Skip the backslash
			p.next()
			// Get the escaped character
			r = p.peek(0)
			if r == '"' {
				result.WriteRune('"')
			} else {
				// For other escape sequences, keep the backslash
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

func (p *ObjectParser) parseInt() (int, error) {
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

func (p *ObjectParser) peek(n int) rune {
	if p.pos+n >= len(p.s) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(p.s[p.pos+n:])
	return r
}

func (p *ObjectParser) next() rune {
	if p.pos >= len(p.s) {
		return 0
	}
	r, w := utf8.DecodeRuneInString(p.s[p.pos:])
	p.pos += w
	p.w = w
	return r
}

func (p *ObjectParser) expect(r rune) bool {
	if p.next() == r {
		return true
	}
	p.pos -= p.w
	return false
}

func (p *ObjectParser) skipSpaces() {
	for {
		r := p.peek(0)
		if r != ' ' && r != '\t' {
			break
		}
		p.next()
	}
}

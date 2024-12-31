package lpc

import (
	"fmt"
	"strconv"
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
	} else if unicode.IsDigit(r) {
		return p.parseInt()
	} else if r == '(' {
		if p.peek(1) == '{' {
			return p.parseList()
		} else if p.peek(1) == '[' {
			return p.parseMap()
		}
	}
	return nil, fmt.Errorf("invalid value type at position %d", p.pos)
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
	}
	return entries, nil
}

func (p *ObjectParser) parseMapEntry() (string, interface{}, error) {
	key, err := p.parseString()
	if err != nil {
		return "", nil, fmt.Errorf("error in map entry: invalid key at position %d", p.pos)
	}
	if !p.expect(':') {
		return "", nil, fmt.Errorf("error in map entry: expected ':' after key at position %d", p.pos)
	}
	p.skipSpaces()
	value, err := p.parseValue()
	if err != nil {
		return "", nil, fmt.Errorf("error in map entry: invalid value at position %d", p.pos)
	}
	if !p.expect(',') {
		return "", nil, fmt.Errorf("error in map entry: expected ',' after value at position %d", p.pos)
	}
	return key, value, nil
}

func (p *ObjectParser) parseList() ([]interface{}, error) {
	if !p.expect('(') || !p.expect('{') {
		return nil, fmt.Errorf("error in list: expected '({' at position %d", p.pos)
	}
	var list []interface{}
	for !p.expect('}') {
		value, err := p.parseValue()
		if err != nil {
			return nil, fmt.Errorf("error in list: invalid value at position %d", p.pos)
		}
		list = append(list, value)
		p.expect(',')
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
	start := p.pos
	for p.peek(0) != '"' {
		if p.peek(0) == 0 {
			return "", fmt.Errorf("error in string: unterminated string at position %d", p.pos)
		}
		p.next()
	}
	if !p.expect('"') {
		return "", fmt.Errorf("error in string: expected '\"' at position %d", p.pos)
	}
	result := p.s[start : p.pos-1]
	return result, nil
}

func (p *ObjectParser) parseInt() (int, error) {
	start := p.pos
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

/* func main() {
	input :=
		`name "Drake"
Str 10
access_map ([1|"drake":([1|"area":([1|"lockers":1,]),]),])`

	p := NewParser(input)
	obj, err := p.ParseObject()
	if err != nil {
		fmt.Printf("Failed to parse the input: %v\n", err)
	} else {
		fmt.Printf("Parsed object: %+v\n", obj)
	}
} */

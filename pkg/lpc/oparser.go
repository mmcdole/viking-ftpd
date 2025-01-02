package lpc

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ObjectParser holds parsing configuration for LPC object format.
// The format is used to store and restore object state in DGD.
type ObjectParser struct {
	strict bool
	file   string // Name of file being parsed
}

// NewObjectParser creates a new parser with the given options.
// In strict mode, any parsing error will stop the entire process.
// In non-strict mode, errors are collected and parsing continues.
func NewObjectParser(strict bool) *ObjectParser {
	return &ObjectParser{
		strict: strict,
	}
}

// SetFile sets the name of the file being parsed
func (p *ObjectParser) SetFile(file string) {
	p.file = file
}

// ParseError represents an error that occurred while parsing a specific line
type ParseError struct {
	File     string // The file being parsed
	Line     int    // The line number where the error occurred
	Position int    // Position within the line where the error occurred
	Err      error  // The specific error encountered
}

func (e *ParseError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("Format error in \"%s\", line %d: %v", e.File, e.Line, e.Err)
	}
	return fmt.Sprintf("Format error at line %d: %v", e.Line, e.Err)
}

// ParseResult contains both the parsed object and any errors encountered.
// In strict mode, Errors will be empty as any error stops parsing.
// In non-strict mode, Errors may contain multiple parsing errors.
type ParseResult struct {
	Object map[string]interface{} // Key-value pairs from the object
	Errors []*ParseError          // Any errors encountered during parsing
}

// LineParser handles parsing of individual lines in LPC object format.
// The format requires:
// - No leading or trailing whitespace
// - Exactly one space between key and value
// - No tabs
// - Keys must be valid identifiers
type LineParser struct {
	s   string // input string
	pos int    // current position in string
	w   int    // width of last rune read
}

// NewLineParser creates a new parser for a single line
func NewLineParser(line string) *LineParser {
	return &LineParser{
		s:   line,
		pos: 0,
		w:   0,
	}
}

// ParseObject parses an LPC object from the input string.
// The input should consist of key-value pairs, one per line.
// Empty lines and lines starting with # are ignored.
// Returns error if input is empty or invalid.
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

	for lineNum, line := range lines {
		// Skip empty lines and comments
		if len(line) == 0 || line[0] == '#' {
			startPos += len(line) + 1 // +1 for newline
			continue
		}

		// Parse key and value
		lp := NewLineParser(line)
		key, value, err := lp.ParseLine()
		if err != nil {
			// Create error with absolute file position
			parseErr := &ParseError{
				File:     p.file,
				Line:     lineNum,
				Position: startPos + lp.pos,
				Err:      err,
			}
			if p.strict {
				return nil, parseErr
			}
			result.Errors = append(result.Errors, parseErr)
		} else {
			result.Object[key] = value
		}

		startPos += len(line) + 1 // +1 for newline
	}

	if len(result.Object) == 0 && len(result.Errors) > 0 {
		return result, fmt.Errorf("no valid entries found")
	}

	return result, nil
}

// ParseLine parses a single line of LPC object format, returning the key and value.
// Format rules:
// - Lines starting with # are treated as comments and skipped
// - Empty lines are skipped
// - No leading or trailing whitespace allowed
// - Exactly one space between key and value
// - No tabs allowed
// - Line must end with newline or EOF
func (p *LineParser) ParseLine() (string, interface{}, error) {
	// Skip comment lines
	if p.peek(0) == '#' {
		return "", nil, nil
	}

	// Skip empty lines
	if p.peek(0) == '\n' || p.peek(0) == 0 {
		return "", nil, nil
	}

	// Leading whitespace is not allowed
	if p.peek(0) == ' ' || p.peek(0) == '\t' {
		return "", nil, fmt.Errorf("leading whitespace not allowed at position %d", p.pos)
	}

	// Parse identifier - must start with letter or underscore
	key, err := p.parseIdentifier()
	if err != nil {
		return "", nil, err
	}

	// Check for exactly one space after key
	if p.peek(0) != ' ' {
		return "", nil, fmt.Errorf("expected single space after key at position %d", p.pos)
	}
	p.next() // consume the single space
	if p.peek(0) == ' ' || p.peek(0) == '\t' {
		return "", nil, fmt.Errorf("multiple spaces or tabs not allowed at position %d", p.pos)
	}

	value, err := p.parseValue()
	if err != nil {
		return "", nil, err
	}

	// Check for trailing whitespace
	r := p.peek(0)
	if r == ' ' || r == '\t' {
		return "", nil, fmt.Errorf("trailing whitespace not allowed at position %d", p.pos)
	}
	if r != '\n' && r != 0 {
		return "", nil, fmt.Errorf("expected newline or end of file at position %d", p.pos)
	}

	return key, value, nil
}

// parseValue parses any valid value type.
// Value types can be:
// - Strings (double-quoted)
// - Integers
// - Floats (with optional hex notation)
// - Arrays
// - Maps
// - nil
func (p *LineParser) parseValue() (interface{}, error) {
	p.skipSpaces()

	r := p.peek(0)
	if r == '"' {
		return p.parseString()
	} else if unicode.IsDigit(r) || r == '-' {
		return p.parseNumber()
	} else if r == '(' {
		// Could be array or map
		if p.peek(1) == '{' {
			return p.parseArray()
		} else if p.peek(1) == '[' {
			return p.parseMap()
		}
		return nil, fmt.Errorf("invalid value starting with '(' at position %d", p.pos)
	} else if r == 'n' {
		// Try parsing nil
		pos := p.pos
		if p.match("nil") {
			// Check that nil is followed by valid terminator
			next := p.peek(0)
			if next == ',' || next == ':' || next == ']' || next == '}' || next == ')' || next == '\n' || next == 0 {
				return nil, nil
			}
			p.pos = pos
		}
		return nil, fmt.Errorf("invalid nil value at position %d", p.pos)
	}

	return nil, fmt.Errorf("invalid value starting with '%c' at position %d", r, p.pos)
}

// parseArray parses an array value.
// Format: ({size|val1,val2,...})
// - Size must match the number of elements
// - Elements are comma-separated with no trailing comma
// - Arrays can be nested
func (p *LineParser) parseArray() ([]interface{}, error) {
	if !p.match("({") {
		return nil, fmt.Errorf("error in array: expected '({' at position %d", p.pos)
	}

	// Parse size
	size, err := p.parseInt()
	if err != nil {
		return nil, fmt.Errorf("error in array: invalid size at position %d: %v", p.pos, err)
	}

	if !p.expect('|') {
		return nil, fmt.Errorf("error in array: expected '|' after size at position %d", p.pos)
	}

	// Parse elements
	elements := make([]interface{}, 0)
	
	// Handle empty array case
	p.skipSpaces()
	if p.peek(0) == '}' && p.peek(1) == ')' {
		p.pos += 2
		if size != 0 {
			return nil, fmt.Errorf("error in array: empty array but size is %d at position %d", size, p.pos)
		}
		return elements, nil
	}

	// Handle empty array with trailing comma
	if p.peek(0) == ',' && p.peek(1) == '}' && p.peek(2) == ')' {
		p.pos += 3
		if size != 0 {
			return nil, fmt.Errorf("error in array: empty array but size is %d at position %d", size, p.pos)
		}
		return elements, nil
	}

	for {
		// Parse element
		element, err := p.parseValue()
		if err != nil {
			return nil, fmt.Errorf("error in array: %v", err)
		}
		elements = append(elements, element)

		p.skipSpaces()
		if p.peek(0) == ',' {
			p.pos++ // consume comma
			p.skipSpaces()
			// Check for trailing comma
			if p.peek(0) == '}' && p.peek(1) == ')' {
				p.pos += 2
				break
			}
			continue
		} else if p.peek(0) == '}' && p.peek(1) == ')' {
			p.pos += 2
			break
		} else {
			return nil, fmt.Errorf("error in array: expected ',' or '})' at position %d", p.pos)
		}
	}

	// Verify size matches number of elements
	if len(elements) > size {
		return nil, fmt.Errorf("error in array: too many elements, expected %d", size)
	} else if len(elements) < size {
		return nil, fmt.Errorf("error in array: too few elements, expected %d", size)
	}

	return elements, nil
}

// parseMap parses a mapping value.
// Format: ([size|key1:val1,key2:val2,...])
// - Size must match the number of entries
// - Entries are comma-separated with no trailing comma
// - Keys can be any valid value type
// - Values can be any valid value type
// - Mappings can be nested
func (p *LineParser) parseMap() (map[string]interface{}, error) {
	if !p.match("([") {
		return nil, fmt.Errorf("error in map: expected '([' at position %d", p.pos)
	}

	// Parse size
	size, err := p.parseInt()
	if err != nil {
		return nil, fmt.Errorf("error in map: invalid size at position %d: %v", p.pos, err)
	}

	if !p.expect('|') {
		return nil, fmt.Errorf("error in map: expected '|' after size at position %d", p.pos)
	}

	// Parse entries
	result := make(map[string]interface{})
	var totalEntries int
	var validEntries int

	// Handle empty map case
	p.skipSpaces()
	if p.peek(0) == ']' && p.peek(1) == ')' {
		p.pos += 2
		if size != 0 {
			return nil, fmt.Errorf("error in map: empty map but size is %d at position %d", size, p.pos)
		}
		return result, nil
	}

	for {
		// Parse key and value
		key, value, skipped, err := p.parseMapEntry()
		if err != nil {
			return nil, err
		}

		if !skipped {
			result[key] = value
			validEntries++
		}
		totalEntries++

		p.skipSpaces()
		if p.peek(0) == ',' {
			p.pos++ // consume comma
			p.skipSpaces()
			// Check for trailing comma
			if p.peek(0) == ']' && p.peek(1) == ')' {
				p.pos += 2
				break
			}
			continue
		} else if p.peek(0) == ']' && p.peek(1) == ')' {
			p.pos += 2
			break
		} else {
			return nil, fmt.Errorf("error in map: expected ',' or '])' at position %d", p.pos)
		}
	}

	// Verify size matches total number of entries (including skipped ones)
	if totalEntries > size {
		return nil, fmt.Errorf("error in map: too many entries, expected %d", size)
	} else if totalEntries < size {
		return nil, fmt.Errorf("error in map: too few entries, expected %d", size)
	}

	// If we have no valid entries but size > 0, that means all entries were skipped
	if validEntries == 0 && size > 0 {
		return make(map[string]interface{}), nil
	}

	return result, nil
}

// ParseFloat parses a float value, optionally with hex notation.
// Format: [-]digits[.digits][=hexdigits]
// The hex part represents the IEEE 754 bits of the float.
func (p *LineParser) parseFloat() (float64, error) {
	start := p.pos

	if p.peek(0) == '-' {
		p.next()
	}

	// Must have at least one digit
	if !unicode.IsDigit(p.peek(0)) {
		return 0, fmt.Errorf("float value must start with a digit at position %d", p.pos)
	}

	// Parse integer part
	for unicode.IsDigit(p.peek(0)) {
		p.next()
	}

	// Parse optional decimal part
	if p.peek(0) == '.' {
		p.next()
		// Must have at least one digit after the decimal point
		if !unicode.IsDigit(p.peek(0)) {
			return 0, fmt.Errorf("float value must have digits after decimal point at position %d", p.pos)
		}
		for unicode.IsDigit(p.peek(0)) {
			p.next()
		}
	}

	// If no decimal point or hex notation, it must have hex notation
	if p.peek(0) != '=' && !strings.Contains(p.s[start:p.pos], ".") {
		return 0, fmt.Errorf("float value must contain a decimal point or hex representation at position %d", p.pos)
	}

	floatStr := p.s[start:p.pos]
	result, err := strconv.ParseFloat(floatStr, 64)
	if err != nil {
		return 0, fmt.Errorf("error in float: invalid number at position %d", p.pos)
	}

	if p.peek(0) == '=' {
		p.next() // skip =
		foundHex := false
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
			foundHex = true
			p.next()
		}
		if !foundHex {
			return 0, fmt.Errorf("invalid hex digits after = at position %d", p.pos)
		}
	}

	return result, nil
}

// ParseString parses a double-quoted string with escape sequences.
// Supported escape sequences:
// - \0  - null character
// - \a  - bell (BEL)
// - \b  - backspace
// - \t  - tab
// - \n  - newline
// - \v  - vertical tab
// - \f  - form feed
// - \r  - carriage return
// - \"  - double quote
// - \\  - backslash
// Any other character after backslash is taken literally.
// Newlines are not allowed in strings.
func (p *LineParser) parseString() (string, error) {
	if !p.match("\"") {
		return "", fmt.Errorf("expected '\"' at position %d", p.pos)
	}

	var b strings.Builder
	for p.pos < len(p.s) {
		r := p.next()
		if r == '"' {
			return b.String(), nil
		}
		if r == '\\' {
			if p.pos >= len(p.s) {
				return "", fmt.Errorf("unterminated string at position %d", p.pos)
			}
			r = p.next()
			switch r {
			case '0':
				b.WriteRune(0)
			case 'a':
				b.WriteRune('\a')
			case 'b':
				b.WriteRune('\b')
			case 't':
				b.WriteRune('\t')
			case 'n':
				b.WriteRune('\n')
			case 'v':
				b.WriteRune('\v')
			case 'f':
				b.WriteRune('\f')
			case 'r':
				b.WriteRune('\r')
			case '"':
				b.WriteRune('"')
			case '\\':
				b.WriteRune('\\')
			default:
				b.WriteRune(r)
			}
			continue
		}
		if r == '\n' {
			return "", fmt.Errorf("newline in string at position %d", p.pos)
		}
		b.WriteRune(r)
	}
	return "", fmt.Errorf("unterminated string at position %d", p.pos)
}

// ParseIdentifier parses an identifier from the input string.
// An identifier consists of letters, digits, and underscores.
// The first character must be a letter.
func (p *LineParser) parseIdentifier() (string, error) {
	start := p.pos
	r := p.next()
	if !unicode.IsLetter(r) {
		return "", fmt.Errorf("identifier must start with a letter at position %d", p.pos)
	}
	for r = p.next(); unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'; r = p.next() {
		if r == 0 {
			return "", fmt.Errorf("unexpected end of input while parsing identifier at position %d", p.pos)
		}
	}
	p.pos -= p.w // back up to last character of identifier
	return p.s[start:p.pos], nil
}

// ParseMapEntry parses a single key:value pair in a mapping.
// Keys can be any valid value type.
// Values can be any valid value type.
func (p *LineParser) parseMapEntry() (string, interface{}, bool, error) {
	p.skipSpaces()

	// Parse key - can be any valid value type
	keyValue, err := p.parseValue()
	if err != nil {
		return "", nil, false, fmt.Errorf("error in map entry: invalid key at position %d: %v", p.pos, err)
	}

	// Convert key to string representation
	var key string
	switch v := keyValue.(type) {
	case string:
		key = v
	case int:
		key = strconv.Itoa(v)
	case float64:
		key = strconv.FormatFloat(v, 'f', -1, 64)
	case []interface{}, map[string]interface{}:
		// Skip array and map keys but count them
		p.skipSpaces()
		if p.peek(0) == ':' {
			p.next() // skip :
			// Parse and discard the value
			if _, err := p.parseValue(); err != nil {
				return "", nil, false, err
			}
		}
		return "", nil, true, nil
	case nil:
		key = "nil"
	default:
		return "", nil, false, fmt.Errorf("error in map entry: unsupported key type %T at position %d", keyValue, p.pos)
	}

	p.skipSpaces()
	if !p.expect(':') {
		return "", nil, false, fmt.Errorf("error in map entry: expected ':' after key at position %d", p.pos)
	}

	// Parse value
	value, err := p.parseValue()
	if err != nil {
		return "", nil, false, err
	}

	return key, value, false, nil
}

// ParseNumber parses either an integer or float value.
// Floats may include hex notation after = sign.
// Format: [-]digits[.digits][=hexdigits]
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

	// If we have hex notation, treat it as a float
	if p.peek(offset) == '=' {
		return p.parseFloat()
	}

	if isFloat {
		return p.parseFloat()
	}
	return p.parseInt()
}

// ParseInt parses an integer from the input string
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

// next returns the next rune and advances the position
func (p *LineParser) next() rune {
	if p.pos >= len(p.s) {
		p.w = 0
		return 0
	}
	r, w := utf8.DecodeRuneInString(p.s[p.pos:])
	p.pos += w
	p.w = w
	return r
}

// Peek returns the rune at the current position without advancing the position
func (p *LineParser) peek(n int) rune {
	if p.pos+n >= len(p.s) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(p.s[p.pos+n:])
	return r
}

// Expect checks if the next rune matches the given rune and advances the position if it does
func (p *LineParser) expect(r rune) bool {
	if p.next() == r {
		return true
	}
	p.pos -= p.w
	return false
}

// SkipSpaces skips any whitespace characters
func (p *LineParser) skipSpaces() {
	for {
		r := p.peek(0)
		if r != ' ' && r != '\t' {
			break
		}
		p.next()
	}
}

// match checks if the next runes match the given string and advances the position if they do
func (p *LineParser) match(s string) bool {
	if p.s[p.pos:p.pos+len(s)] == s {
		p.pos += len(s)
		return true
	}
	return false
}

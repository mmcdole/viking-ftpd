package lpc

import (
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestParser_ParseObject(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name:  "Single Line",
			input: `name "Drake"`,
			expected: map[string]interface{}{
				"name": "Drake",
			},
			wantErr: false,
		},
		{
			name:  "Multiple Lines",
			input: "name \"Drake\"\nStr 10",
			expected: map[string]interface{}{
				"name": "Drake",
				"Str":  10,
			},
			wantErr: false,
		},
		{
			name:  "Nested Map",
			input: `access_map ([1|"drake":([1|"area":([1|"lockers":1,]),]),])`,
			expected: map[string]interface{}{
				"access_map": map[string]interface{}{
					"drake": map[string]interface{}{
						"area": map[string]interface{}{
							"lockers": 1,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "Array of Numbers",
			input: `boards ({3|1,2,3,})`,
			expected: map[string]interface{}{
				"boards": []interface{}{1, 2, 3},
			},
			wantErr: false,
		},
		{
			name:  "Floating Point Numbers",
			input: "army_change 0.19979999972566\n",
			expected: map[string]interface{}{
				"army_change": 0.19979999972566,
			},
			wantErr: false,
		},
		{
			name:  "Empty Map",
			input: `Admin ([0|])`,
			expected: map[string]interface{}{
				"Admin": map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name:  "Negative Integer",
			input: `alignment -39`,
			expected: map[string]interface{}{
				"alignment": -39,
			},
			wantErr: false,
		},
		{
			name:  "Escaped Characters",
			input: "long_desc \"this is a test\\nwith newline\\n\"\n",
			expected: map[string]interface{}{
				"long_desc": "this is a test\\nwith newline\\n",
			},
			wantErr: false,
		},
		{
			name:  "Mixed Types in Map",
			input: `tags ({2|"test",10,})`,
			expected: map[string]interface{}{
				"tags": []interface{}{"test", 10},
			},
			wantErr: false,
		},
		{
			name:  "Special Characters",
			input: "plan \"Still round the corner may wait\\nEast of the Sun\"\n",
			expected: map[string]interface{}{
				"plan": "Still round the corner may wait\\nEast of the Sun",
			},
			wantErr: false,
		},
		{
			name:  "Complex Nested Structure",
			input: `m_property ([3|"COLOURS":([2|"board-subject":"","exits":"%^L_GREEN%^",]),"bn_boards":({3|1,2,3,}),"combat_status":([2|"block":10,"counter":10,]),])`,
			expected: map[string]interface{}{
				"m_property": map[string]interface{}{
					"COLOURS": map[string]interface{}{
						"board-subject": "",
						"exits":         "%^L_GREEN%^",
					},
					"bn_boards": []interface{}{1, 2, 3},
					"combat_status": map[string]interface{}{
						"block":   10,
						"counter": 10,
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "Multiple Value Types",
			input: `mixed_map ([4|"name":"drake","level":10,"skills":({2|"sword","bow",}),"stats":([2|"str":10,"dex":12,]),])`,
			expected: map[string]interface{}{
				"mixed_map": map[string]interface{}{
					"name":  "drake",
					"level": 10,
					"skills": []interface{}{
						"sword",
						"bow",
					},
					"stats": map[string]interface{}{
						"str": 10,
						"dex": 12,
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "Empty String",
			input:   ``,
			expected: nil,
			wantErr: true,
		},
		{
			name:  "Float with Hex",
			input: `army_change 0.19979999972566=3ffc9930be04800000000000`,
			expected: map[string]interface{}{
				"army_change": float64(0.19979999972566),
			},
			wantErr: false,
		},
		{
			name:  "Escaped Quotes",
			input: `key1:"value with \"escaped\" quotes"
key2:"another \"quoted\" string"`,
			expected: map[string]interface{}{
				"key1": "value with \"escaped\" quotes",
				"key2": "another \"quoted\" string",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewObjectParser(tt.input)
			got, err := p.ParseObject()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.ParseObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Parser.ParseObject() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseDrakeCharacter(t *testing.T) {
	data, err := os.ReadFile("../../resources/drake.o.txt")
	if err != nil {
		t.Fatalf("Failed to read drake.o.txt: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		p := NewObjectParser(line)
		_, err := p.ParseObject()
		if err != nil {
			// Extract position from error message
			pos := 0
			if matches := regexp.MustCompile(`position (\d+)`).FindStringSubmatch(err.Error()); matches != nil {
				pos, _ = strconv.Atoi(matches[1])
			}
			
			start := pos - 10
			if start < 0 {
				start = 0
			}
			end := pos + 10
			if end > len(line) {
				end = len(line)
			}
			t.Errorf("Failed to parse line %d: %q\nContext around position %d: %q\nError: %v", 
				i+1, line, pos, line[start:end], err)
			// Stop at first error
			break
		}
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "string",
			input:    `"hello"`,
			expected: "hello",
		},
		{
			name:     "integer",
			input:    "42",
			expected: 42,
		},
		{
			name:     "float",
			input:    "3.14",
			expected: float64(3.14),
		},
		{
			name:     "float with hex",
			input:    "0.19979999972566=3ffc9930be04800000000000",
			expected: float64(0.19979999972566),
		},
		{
			name:     "empty list",
			input:    "({0|})",
			expected: []interface{}{},
		},
		{
			name:     "list with values",
			input:    `({2|"hello",42})`,
			expected: []interface{}{"hello", 42},
		},
		{
			name:     "empty map",
			input:    "([0|])",
			expected: map[string]interface{}{},
		},
		{
			name:     "map with entries",
			input:    `([2|"key":"value","num":42])`,
			expected: map[string]interface{}{"key": "value", "num": 42},
		},
		{
			name:    "invalid value",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewObjectParser(tt.input)
			got, err := p.parseValue()
			if (err != nil) != tt.wantErr {
				t.Errorf("parseValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseValue() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseMapWithEscapedQuotes(t *testing.T) {
	input := `1:"normal string"
2:"string with \"escaped\" quotes"
3:"string with \"multiple\" \"escaped\" quotes"`
	
	parser := NewObjectParser(input)
	result, err := parser.ParseObject()
	
	if err != nil {
		t.Errorf("Failed to parse map with escaped quotes: %v", err)
	}
	
	expected := map[string]interface{}{
		"1": "normal string",
		"2": "string with \"escaped\" quotes",
		"3": "string with \"multiple\" \"escaped\" quotes",
	}
	
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v but got %v", expected, result)
	}
}

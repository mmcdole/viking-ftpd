package lpc

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestObjectParser tests the high-level object parsing functionality
func TestObjectParser(t *testing.T) {
	t.Run("File Structure", func(t *testing.T) {
		tests := []struct {
			name    string
			input   string
			want    map[string]interface{}
			wantErr bool
		}{
			{
				name:    "Empty Input",
				input:   "",
				wantErr: true,
			},
			{
				name:  "Single Line",
				input: `name "Drake"`,
				want: map[string]interface{}{
					"name": "Drake",
				},
			},
			{
				name: "Multiple Lines",
				input: `name "Drake"
level 30
title "wizard"`,
				want: map[string]interface{}{
					"name":  "Drake",
					"level": 30,
					"title": "wizard",
				},
			},
			{
				name: "Empty Lines",
				input: `name "Drake"

level 30

title "wizard"`,
				want: map[string]interface{}{
					"name":  "Drake",
					"level": 30,
					"title": "wizard",
				},
			},
			{
				name: "Comments",
				input: `# This is a comment
name "Drake"
# Another comment
level 30`,
				want: map[string]interface{}{
					"name":  "Drake",
					"level": 30,
				},
			},
			{
				name: "Comment and Empty Lines",
				input: `# Header comment

name "Drake"
# Mid comment

level 30
# End comment`,
				want: map[string]interface{}{
					"name":  "Drake",
					"level": 30,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := NewObjectParser(false)
				got, err := p.ParseObject(tt.input)
				if (err != nil) != tt.wantErr {
					t.Errorf("ParseObject() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && !reflect.DeepEqual(got.Object, tt.want) {
					t.Errorf("ParseObject() got = %v, want %v", got.Object, tt.want)
				}
			})
		}
	})

	t.Run("Error Handling", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			strict   bool
			wantErr  bool
			wantErrs int // number of errors in non-strict mode
		}{
			{
				name: "Multiple Errors Strict Mode",
				input: `name = Drake
age = 30
title = "wizard"`,
				strict:   true,
				wantErr:  true,
				wantErrs: 3,
			},
			{
				name: "Multiple Errors Non-Strict Mode",
				input: `name = Drake
valid_line "test"
age = 30`,
				strict:   false,
				wantErr:  false,
				wantErrs: 2,
			},
			{
				name: "Mixed Valid and Invalid",
				input: `name "Drake"
age = 30
level 5`,
				strict:   false,
				wantErr:  false,
				wantErrs: 1,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := NewObjectParser(tt.strict)
				got, err := p.ParseObject(tt.input)
				if tt.strict {
					if (err != nil) != tt.wantErr {
						t.Errorf("ParseObject() error = %v, wantErr %v", err, tt.wantErr)
					}
				} else {
					if len(got.Errors) != tt.wantErrs {
						t.Errorf("ParseObject() got %d errors, want %d", len(got.Errors), tt.wantErrs)
					}
				}
			})
		}
	})
}

// TestLineParser tests the mid-level line parsing functionality
func TestLineParser(t *testing.T) {
	t.Run("Line Structure", func(t *testing.T) {
		tests := []struct {
			name    string
			line    string
			wantKey string
			wantVal interface{}
			wantErr bool
		}{
			{
				name:    "Basic Key-Value",
				line:    `name "Drake"`,
				wantKey: "name",
				wantVal: "Drake",
			},
			{
				name:    "Empty Line",
				line:    "",
				wantKey: "",
				wantVal: nil,
			},
			{
				name:    "Comment Line",
				line:    `# This is a comment`,
				wantKey: "",
				wantVal: nil,
			},
			{
				name:    "Missing Value",
				line:    "name",
				wantErr: true,
			},
			{
				name:    "Invalid Key",
				line:    "123name value",
				wantErr: true,
			},
			{
				name:    "Empty Line",
				line:    "",
				wantKey: "",
				wantVal: nil,
			},
			{
				name:    "Line With Only Spaces",
				line:    "   ",
				wantErr: true,
			},
			{
				name:    "Line With Only Tab",
				line:    "\t",
				wantErr: true,
			},
			{
				name:    "Line With Mixed Whitespace",
				line:    " \t  ",
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := NewLineParser(tt.line)
				key, val, err := p.ParseLine()
				if (err != nil) != tt.wantErr {
					t.Errorf("ParseLine() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if key != tt.wantKey {
					t.Errorf("ParseLine() key = %v, want %v", key, tt.wantKey)
				}
				if !reflect.DeepEqual(val, tt.wantVal) {
					t.Errorf("ParseLine() val = %v, want %v", val, tt.wantVal)
				}
			})
		}
	})

	t.Run("Whitespace Handling", func(t *testing.T) {
		tests := []struct {
			name    string
			line    string
			wantKey string
			wantVal interface{}
			wantErr bool
		}{
			{
				name:    "Single Space After Key",
				line:    "name \"Drake\"",
				wantKey: "name",
				wantVal: "Drake",
			},
			{
				name:  "Leading Space Not Allowed",
				line:  "   name \"Drake\"",
				wantErr: true,
			},
			{
				name:  "Multiple Spaces Not Allowed",
				line:  "name    \"Drake\"",
				wantErr: true,
			},
			{
				name:  "Trailing Space Not Allowed",
				line:  "name \"Drake\"   ",
				wantErr: true,
			},
			{
				name:  "Tab Not Allowed",
				line:  "name\t\"Drake\"",
				wantErr: true,
			},
			{
				name:    "Comment Line",
				line:    "# This is a comment",
				wantKey: "",
				wantVal: nil,
			},
			{
				name:  "Leading Space In Comment Not Allowed",
				line:  "   # This is a comment",
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := NewLineParser(tt.line)
				key, val, err := p.ParseLine()
				if (err != nil) != tt.wantErr {
					t.Errorf("ParseLine() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if key != tt.wantKey {
					t.Errorf("ParseLine() key = %v, want %v", key, tt.wantKey)
				}
				if !reflect.DeepEqual(val, tt.wantVal) {
					t.Errorf("ParseLine() val = %v, want %v", val, tt.wantVal)
				}
			})
		}
	})

	t.Run("String Escapes", func(t *testing.T) {
		tests := []struct {
			name    string
			input   string
			want    interface{}
			wantErr bool
		}{
			{
				name:  "Basic String",
				input: `"hello"`,
				want:  "hello",
			},
			{
				name:  "String with Escapes",
				input: `"hello\nworld\t!"`,
				want:  "hello\nworld\t!",
			},
			{
				name:  "Escaped Backslash",
				input: `"C:\\path\\to\\file"`,
				want:  `C:\path\to\file`,
			},
			{
				name:  "Null Character",
				input: `"hello\0world"`,
				want:  "hello\x00world",
			},
			{
				name:  "Bell",
				input: `"alert\a"`,
				want:  "alert\a",
			},
			{
				name:  "Backspace",
				input: `"back\bspace"`,
				want:  "back\bspace",
			},
			{
				name:  "Tab",
				input: `"hello\tworld"`,
				want:  "hello\tworld",
			},
			{
				name:  "Newline",
				input: `"hello\nworld"`,
				want:  "hello\nworld",
			},
			{
				name:  "Vertical Tab",
				input: `"form\vfeed"`,
				want:  "form\vfeed",
			},
			{
				name:  "Form Feed",
				input: `"form\ffeed"`,
				want:  "form\ffeed",
			},
			{
				name:  "Carriage Return",
				input: `"hello\rworld"`,
				want:  "hello\rworld",
			},
			{
				name:  "Multiple Escapes",
				input: `"line1\nline2\tcolumn2\nline3"`,
				want:  "line1\nline2\tcolumn2\nline3",
			},
			{
				name:  "Unknown Escape Sequence",
				input: `"hello\zworld"`,
				want:  "hellozworld", // DGD behavior: unknown escapes are taken literally
			},
			{
				name:    "Unterminated String",
				input:   `"hello`,
				wantErr: true,
			},
			{
				name:    "Unterminated String with Escape",
				input:   `"hello\`,
				wantErr: true,
			},
			{
				name:    "Newline in String",
				input:   "\"hello\nworld\"",
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := NewLineParser(tt.input)
				got, err := p.parseValue()
				if (err != nil) != tt.wantErr {
					t.Errorf("parseValue() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
					t.Errorf("parseValue() got = %v, want %v", got, tt.want)
				}
			})
		}
	})
}

// TestValueParsing tests the low-level value parsing functionality
func TestValueParsing(t *testing.T) {
	t.Run("Basic Types", func(t *testing.T) {
		tests := []struct {
			name    string
			input   string
			want    interface{}
			wantErr bool
		}{
			{
				name:  "String",
				input: `"hello"`,
				want:  "hello",
			},
			{
				name:  "String with Escapes",
				input: `"hello\nworld\t!"`,
				want:  "hello\nworld\t!",
			},
			{
				name:  "Integer",
				input: "42",
				want:  42,
			},
			{
				name:  "Negative Integer",
				input: "-42",
				want:  -42,
			},
			{
				name:  "Float",
				input: "3.14",
				want:  3.14,
			},
			{
				name:  "Float with Hex",
				input: "0.19979999972566=3ffc9930be04800000000000",
				want:  0.19979999972566,
			},
			{
				name:  "Float with Hex Only",
				input: "1=3ff0000000000000",
				want:  1.0,
			},
			{
				name:  "Float with Both Decimal and Hex",
				input: "1.0=3ff0000000000000",
				want:  1.0,
			},
			{
				name:  "Zero with Hex",
				input: "0=0000000000000000",
				want:  0.0,
			},
			{
				name:  "Negative with Hex",
				input: "-1=bff0000000000000",
				want:  -1.0,
			},
			{
				name:  "Nil Value",
				input: "nil",
				want:  nil,
			},
			{
				name:    "Invalid Nil Value",
				input:   "nill",
				wantErr: true,
			},
			{
				name:    "Invalid Nil Case",
				input:   "NIL",
				wantErr: true,
			},
			{
				name:  "Negative Float",
				input: "-3.14",
				want:  -3.14,
			},
			{
				name:  "Negative Float with Hex",
				input: "-0.19979999972566=3ffc9930be04800000000000",
				want:  -0.19979999972566,
			},
			{
				name:    "Invalid Float Hex",
				input:   "1.0=xyz",
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := NewLineParser(tt.input)
				got, err := p.parseValue()
				if (err != nil) != tt.wantErr {
					t.Errorf("parseValue() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
					t.Errorf("parseValue() got = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("Arrays", func(t *testing.T) {
		tests := []struct {
			name    string
			input   string
			want    interface{}
			wantErr bool
		}{
			{
				name:  "Empty Array",
				input: "({0|})",
				want:  []interface{}{},
			},
			{
				name:  "Empty Array with Trailing Comma",
				input: "({0|,})",
				want:  []interface{}{},
			},
			{
				name:  "Simple Array",
				input: `({3|1,2,3})`,
				want:  []interface{}{1, 2, 3},
			},
			{
				name:  "Simple Array with Trailing Comma",
				input: `({3|1,2,3,})`,
				want:  []interface{}{1, 2, 3},
			},
			{
				name:  "Mixed Type Array",
				input: `({3|"hello",42,3.14})`,
				want:  []interface{}{"hello", 42, 3.14},
			},
			{
				name:  "Nested Array",
				input: `({2|({2|1,2}),({2|3,4})})`,
				want:  []interface{}{[]interface{}{1, 2}, []interface{}{3, 4}},
			},
			{
				name:  "Array with Nil",
				input: `({3|nil,1,nil})`,
				want:  []interface{}{nil, 1, nil},
			},
			{
				name:    "Array Size Mismatch More",
				input:   `({2|1,2,3})`,
				wantErr: true,
			},
			{
				name:    "Array Size Mismatch Less",
				input:   `({3|1,2})`,
				wantErr: true,
			},
			{
				name:    "Malformed Array Start",
				input:   `{3|1,2,3})`,
				wantErr: true,
			},
			{
				name:    "Malformed Array End",
				input:   `({3|1,2,3`,
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := NewLineParser(tt.input)
				got, err := p.parseValue()
				if (err != nil) != tt.wantErr {
					t.Errorf("parseValue() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
					t.Errorf("parseValue() got = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("Maps", func(t *testing.T) {
		tests := []struct {
			name    string
			input   string
			want    interface{}
			wantErr bool
		}{
			{
				name:  "Empty Map",
				input: "([0|])",
				want:  map[string]interface{}{},
			},
			{
				name:  "Simple Map",
				input: `([2|"name":"Drake","age":30])`,
				want:  map[string]interface{}{"name": "Drake", "age": 30},
			},
			{
				name:  "Nested Map",
				input: `([1|"stats":([2|"str":18,"dex":16])])`,
				want: map[string]interface{}{
					"stats": map[string]interface{}{
						"str": 18,
						"dex": 16,
					},
				},
			},
			{
				name:  "Deep Nested Map",
				input: `([1|"drake":([1|"area":([1|"lockers":1])])])`,
				want: map[string]interface{}{
					"drake": map[string]interface{}{
						"area": map[string]interface{}{
							"lockers": 1,
						},
					},
				},
			},
			{
				name:  "Map with Mixed Types",
				input: `([3|"name":"Drake","stats":([2|"str":18,"dex":16]),"inventory":({2|"sword","shield"})])`,
				want: map[string]interface{}{
					"name": "Drake",
					"stats": map[string]interface{}{
						"str": 18,
						"dex": 16,
					},
					"inventory": []interface{}{"sword", "shield"},
				},
			},
			{
				name:  "Skip Array Key Keep String Key",
				input: `([2|({2|1,2}):"skipped","valid":"kept"])`,
				want: map[string]interface{}{
					"valid": "kept",
				},
			},
			{
				name:  "Skip Map Key Keep Int Key",
				input: `([2|([1|"inner":"value"]):"skipped",42:"kept"])`,
				want: map[string]interface{}{
					"42": "kept",
				},
			},
			{
				name:  "Multiple Skip Types",
				input: `([4|({1|1}):"skip1",([1|"x":"y"]):"skip2","str":"keep1",123:"keep2"])`,
				want: map[string]interface{}{
					"str": "keep1",
					"123": "keep2",
				},
			},
			{
				name:  "Map with Nil",
				input: `([2|nil:"value","key":nil])`,
				want: map[string]interface{}{
					"nil": "value",
					"key": nil,
				},
			},
			{
				name:  "Map with Number Keys",
				input: `([3|42:"int",3.14:"float",-1:"negative"])`,
				want: map[string]interface{}{
					"42":   "int",
					"3.14": "float",
					"-1":   "negative",
				},
			},
			{
				name:    "Map Size Mismatch More",
				input:   `([1|"a":1,"b":2])`,
				wantErr: true,
			},
			{
				name:    "Map Size Mismatch Less",
				input:   `([2|"a":1])`,
				wantErr: true,
			},
			{
				name:    "Malformed Map Start",
				input:   `[2|"a":1,"b":2])`,
				wantErr: true,
			},
			{
				name:    "Malformed Map End",
				input:   `([2|"a":1,"b":2`,
				wantErr: true,
			},
			{
				name:    "Invalid Key-Value Separator",
				input:   `([1|"key"="value"])`,
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := NewLineParser(tt.input)
				got, err := p.parseValue()
				if (err != nil) != tt.wantErr {
					t.Errorf("parseValue() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
					t.Errorf("parseValue() got = %v, want %v", got, tt.want)
				}
			})
		}
	})
}

func TestNestedStructures(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		errMsg   string
		expected interface{}
	}{
		{
			name:  "Map with array value",
			input: "([1|\"key\":({2|1,2})])",
			expected: map[string]interface{}{
				"key": []interface{}{1, 2},
			},
		},
		{
			name:  "Map with nested map",
			input: "([1|\"outer\":([2|\"inner1\":1,\"inner2\":2])])",
			expected: map[string]interface{}{
				"outer": map[string]interface{}{
					"inner1": 1,
					"inner2": 2,
				},
			},
		},
		{
			name:    "Map with array value - wrong array size",
			input:   "([1|\"key\":({2|1,2,3})])",
			wantErr: true,
			errMsg:  "error in array: too many elements, expected 2",
		},
		{
			name:    "Map with array value - wrong map size",
			input:   "([1|\"key1\":({2|1,2}),\"key2\":({2|1,2})])",
			wantErr: true,
			errMsg:  "error in map: too many entries, expected 1",
		},
		{
			name:    "Nested map with wrong inner size",
			input:   "([1|\"outer\":([2|\"inner1\":1,\"inner2\":2,\"inner3\":3])])",
			wantErr: true,
			errMsg:  "error in map: too many entries, expected 2",
		},
		{
			name:    "Deeply nested array with too few elements",
			input:   "([1|\"level1\":([1|\"level2\":({3|1,2})])])",
			wantErr: true,
			errMsg:  "error in array: too few elements, expected 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewLineParser(tt.input)
			got, err := p.parseValue()
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseValue() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("parseValue() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("parseValue() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseValue() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestCharacterFiles tests parsing of actual character files
func TestParseAllCharacters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping character file tests in short mode. Remove -short to run")
	}

	files, err := filepath.Glob("../../resources/characters/*/*")
	if err != nil {
		t.Fatal(err)
	}

	logFile, err := os.Create("parse_failures.log")
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()

	fmt.Fprintf(logFile, "\nCharacter File Parse Failures - %s\n\n", time.Now().Format(time.RFC3339))

	totalFiles := 0
	failures := 0
	failingChars := make([]string, 0)

	for _, file := range files {
		if filepath.Ext(file) != ".o" {
			continue
		}
		totalFiles++

		t.Run(filepath.Base(file), func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}

			p := NewObjectParser(false)
			result, err := p.ParseObject(string(data))
			if err != nil {
				t.Fatal(err)
			}

			if len(result.Errors) > 0 {
				failures++
				charName := filepath.Base(file)
				errorMsg := fmt.Sprintf("%s: %v", charName, result.Errors[0])
				failingChars = append(failingChars, errorMsg)

				fmt.Fprintf(logFile, "File: %s\n", file)
				for _, parseErr := range result.Errors {
					fmt.Fprintf(logFile, "Line: %d\n", parseErr.Line)
					fmt.Fprintf(logFile, "Error: %v\n\n", parseErr.Err)
				}
			}
		})
	}

	t.Logf("\nSummary:\nTotal files processed: %d\nTotal failures: %d", totalFiles, failures)
	if failures > 0 {
		t.Logf("\nFailing characters:\n%s", strings.Join(failingChars, "\n"))
		t.Errorf("Failed to parse %d out of %d character files", failures, totalFiles)
	}
}

func TestParseDiosO(t *testing.T) {
	data, err := os.ReadFile("../../resources/characters/d/dios.o")
	if err != nil {
		t.Fatalf("Failed to read dios.o: %v", err)
	}

	parser := NewObjectParser(false)
	parser.SetFile("dios.o")
	result, err := parser.ParseObject(string(data))
	if err != nil {
		t.Errorf("Failed to parse dios.o: %v", err)
	}

	if result != nil && len(result.Errors) > 0 {
		t.Logf("Found %d errors", len(result.Errors))
		for _, err := range result.Errors {
			line := strings.Split(string(data), "\n")[err.Line]
			t.Logf("Line: %q", line)
			t.Logf("Line length: %d", len(line))
			t.Logf("Error: %v\n", err.Err)

			// Print the problematic value
			lineParser := NewLineParser(line)
			_, value, _ := lineParser.ParseLine()
			t.Logf("Value: %#v\n", value)

			// Print characters around error position
			if err.Position < len(line) {
				start := err.Position - 20
				if start < 0 {
					start = 0
				}
				end := err.Position + 20
				if end > len(line) {
					end = len(line)
				}
				t.Logf("Context around position %d: %q", err.Position, line[start:end])
				t.Logf("Character at position: %q", string(line[err.Position]))
			} else {
				t.Logf("Error position %d is beyond line length %d", err.Position, len(line))
			}
		}
	}
}

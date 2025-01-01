package lpc

import (
	"reflect"
	"testing"
	"fmt"
	"os"
	"time"
	"strings"
	"path/filepath"
)

func TestParser_ParseObject(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name:  "Single Line",
			input: `name "Drake"`,
			want: map[string]interface{}{
				"name": "Drake",
			},
			wantErr: false,
		},
		{
			name:  "Multiple Lines",
			input: "name \"Drake\"\nStr 10",
			want: map[string]interface{}{
				"name": "Drake",
				"Str":  10,
			},
			wantErr: false,
		},
		{
			name:  "Nested Map",
			input: `access_map ([1|"drake":([1|"area":([1|"lockers":1,]),]),])`,
			want: map[string]interface{}{
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
			want: map[string]interface{}{
				"boards": []interface{}{1, 2, 3},
			},
			wantErr: false,
		},
		{
			name:  "Floating Point Numbers",
			input: "army_change 0.19979999972566\n",
			want: map[string]interface{}{
				"army_change": 0.19979999972566,
			},
			wantErr: false,
		},
		{
			name:  "Empty Map",
			input: `Admin ([0|])`,
			want: map[string]interface{}{
				"Admin": map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name:  "Negative Integer",
			input: `alignment -39`,
			want: map[string]interface{}{
				"alignment": -39,
			},
			wantErr: false,
		},
		{
			name:  "Escaped Characters",
			input: "long_desc \"this is a test\\nwith newline\\n\"\n",
			want: map[string]interface{}{
				"long_desc": "this is a test\\nwith newline\\n",
			},
			wantErr: false,
		},
		{
			name:  "Mixed Types in Map",
			input: `tags ({2|"test",10,})`,
			want: map[string]interface{}{
				"tags": []interface{}{"test", 10},
			},
			wantErr: false,
		},
		{
			name:  "Special Characters",
			input: "plan \"Still round the corner may wait\\nEast of the Sun\"\n",
			want: map[string]interface{}{
				"plan": "Still round the corner may wait\\nEast of the Sun",
			},
			wantErr: false,
		},
		{
			name:  "Complex Nested Structure",
			input: `m_property ([3|"COLOURS":([2|"board-subject":"","exits":"%^L_GREEN%^",]),"bn_boards":({3|1,2,3,}),"combat_status":([2|"block":10,"counter":10,]),])`,
			want: map[string]interface{}{
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
			want: map[string]interface{}{
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
			want:    nil,
			wantErr: true,
		},
		{
			name:  "Float with Hex",
			input: `army_change 0.19979999972566=3ffc9930be04800000000000`,
			want: map[string]interface{}{
				"army_change": float64(0.19979999972566),
			},
			wantErr: false,
		},
		{
			name:  "Escaped Quotes",
			input: `key1:"value with \"escaped\" quotes"
key2:"another \"quoted\" string"`,
			want: map[string]interface{}{
				"key1": "value with \"escaped\" quotes",
				"key2": "another \"quoted\" string",
			},
			wantErr: false,
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
			if !reflect.DeepEqual(got.Object, tt.want) {
				t.Errorf("ParseObject() got = %v, want %v", got.Object, tt.want)
			}
		})
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    interface{}
		wantErr bool
	}{
		{
			name:    "string",
			input:   `"hello"`,
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "integer",
			input:   "42",
			want:    42,
			wantErr: false,
		},
		{
			name:    "float",
			input:   "3.14",
			want:    float64(3.14),
			wantErr: false,
		},
		{
			name:    "float with hex",
			input:   "0.19979999972566=3ffc9930be04800000000000",
			want:    float64(0.19979999972566),
			wantErr: false,
		},
		{
			name:    "empty list",
			input:   "({0|})",
			want:    []interface{}{},
			wantErr: false,
		},
		{
			name:    "list with values",
			input:   `({2|"hello",42})`,
			want:    []interface{}{"hello", 42},
			wantErr: false,
		},
		{
			name:    "empty map",
			input:   "([0|])",
			want:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name:    "map with entries",
			input:   `([2|"key":"value","num":42])`,
			want:    map[string]interface{}{"key": "value", "num": 42},
			wantErr: false,
		},
		{
			name:    "invalid value",
			input:   "invalid",
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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseValue() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseAllCharacters(t *testing.T) {
	files, err := filepath.Glob("../../resources/characters/*/*")
	if err != nil {
		t.Fatal(err)
	}

	// Create or truncate the parse failures log
	logFile, err := os.Create("parse_failures.log")
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()

	// Write header
	fmt.Fprintf(logFile, "\nCharacter File Parse Failures - %s\n\n\n", time.Now().Format(time.RFC3339))

	totalFiles := 0
	failures := 0
	failingChars := make([]string, 0)

	for _, file := range files {
		if filepath.Ext(file) != ".o" {
			continue
		}
		totalFiles++

		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		p := NewObjectParser(false) // non-strict mode
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
			// Write failure to log file
			for _, parseErr := range result.Errors {
				fmt.Fprintf(logFile, "Line: %q\n", parseErr.Line)
				fmt.Fprintf(logFile, "Error: %v\n\n", parseErr.Err)
			}
		}
	}

	t.Logf("\nSummary:\nTotal files processed: %d\nTotal failures: %d", totalFiles, failures)
	if failures > 0 {
		t.Logf("\nFailing characters:\n%s", strings.Join(failingChars, "\n"))
		t.Errorf("Failed to parse %d out of %d character files. See details above.", failures, totalFiles)
	}
}

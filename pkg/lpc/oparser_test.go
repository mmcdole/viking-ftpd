package lpc

import (
	"reflect"
	"testing"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewObjectParser(tt.input)
			got, err := p.ParseObject()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.ParseObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.ParseObject() = %v, want %v", got, tt.want)
			}
		})
	}
}

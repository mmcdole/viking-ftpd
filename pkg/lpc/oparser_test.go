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
			name:  "Nested Map with Multiple Lines",
			input: "name \"Drake\"\nStr 10\naccess_map ([1|\"drake\":([1|\"area\":([1|\"lockers\":1,]),]),])",
			want: map[string]interface{}{
				"name": "Drake",
				"Str":  10,
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

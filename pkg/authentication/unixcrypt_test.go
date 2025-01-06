package authentication

import "testing"

func TestUnixCrypt(t *testing.T) {
	tests := []struct {
		name     string
		password string
		hash     string
		wantErr  bool
	}{
		{
			name:     "valid password matches hash",
			password: "testpassword123",
			hash:     "tek4edTZE898g",
			wantErr:  false,
		},
		{
			name:     "incorrect password with valid hash format",
			password: "wrongpassword",
			hash:     "tek4edTZE898g",
			wantErr:  true,
		},
		{
			name:     "malformed hash format",
			password: "testpassword123",
			hash:     "x",
			wantErr:  true,
		},
	}

	hasher := NewUnixCrypt()

	t.Run("hash generation", func(t *testing.T) {
		got, err := hasher.Hash(tests[0].password)
		if err != nil {
			t.Fatalf("Hash() error = %v", err)
		}
		if got != tests[0].hash {
			t.Errorf("Hash() = %q, want %q", got, tests[0].hash)
		}
	})

	t.Run("password verification", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := hasher.VerifyPassword(tt.password, tt.hash)
				if (err != nil) != tt.wantErr {
					t.Errorf("VerifyPassword() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})
}

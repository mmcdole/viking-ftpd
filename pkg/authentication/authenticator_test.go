package authentication

import (
	"errors"
	"testing"

	"github.com/mmcdole/viking-ftpd/pkg/users"
	"github.com/stretchr/testify/assert"
)

// mockSource implements users.Source for testing
type mockSource struct {
	users map[string]*users.User
}

func newMockSource() *mockSource {
	return &mockSource{
		users: make(map[string]*users.User),
	}
}

func (s *mockSource) LoadUser(username string) (*users.User, error) {
	user, ok := s.users[username]
	if !ok {
		return nil, users.ErrUserNotFound
	}
	return user, nil
}

func (s *mockSource) addUser(username, passwordHash string, level int) {
	s.users[username] = &users.User{
		Username:     username,
		PasswordHash: passwordHash,
		Level:       level,
	}
}

// mockVerifier implements PasswordVerifier for testing
type mockVerifier struct {
	expectedHash     string
	expectedPassword string
}

func (m *mockVerifier) VerifyPassword(password, hashedPassword string) error {
	if hashedPassword == m.expectedHash && password == m.expectedPassword {
		return nil
	}
	return errors.New("password mismatch")
}

func TestAuthenticator_Authenticate(t *testing.T) {
	source := newMockSource()
	verifier := &mockVerifier{
		expectedHash:     "hashedpass123",
		expectedPassword: "testpass123",
	}

	// Add test user with expected hash
	source.addUser("user1", "hashedpass123", 1)

	auth := NewAuthenticator(source, verifier)

	tests := []struct {
		name          string
		username      string
		password      string
		wantErr       error
		wantLevel     int
	}{
		{
			name:      "valid credentials",
			username:  "user1",
			password:  "testpass123",
			wantErr:   nil,
			wantLevel: 1,
		},
		{
			name:     "invalid password",
			username: "user1",
			password: "wrongpass",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name:     "user not found",
			username: "nonexistent",
			password: "testpass123",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name:     "user not found with wrong password",
			username: "nonexistent",
			password: "wrongpass",
			wantErr:  ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := auth.Authenticate(tt.username, tt.password)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.wantLevel, user.Level)
			}
		})
	}
}

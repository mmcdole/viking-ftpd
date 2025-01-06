package authorization

import (
	"github.com/mmcdole/viking-ftpd/pkg/users"
)

// TestUser represents a test user with a name and level
type TestUser struct {
	Name  string
	Level int
}

// NewTestUserSource creates a memory source with the given test users
func NewTestUserSource(testUsers []TestUser) users.Source {
	source := users.NewMemorySource()
	for _, u := range testUsers {
		source.AddUser(&users.User{
			Username: u.Name,
			Level:   u.Level,
		})
	}
	return source
}

// NewTestAccessSource creates an access source with the given permissions
func NewTestAccessSource(perms map[string]Permission) AccessSource {
	return &testAccessSource{perms: perms}
}

type testAccessSource struct {
	perms map[string]Permission
}

func (s *testAccessSource) LoadRawData() (map[string]interface{}, error) {
	data := make(map[string]interface{})
	for path, perm := range s.perms {
		data[path] = int(perm)
	}
	return data, nil
}

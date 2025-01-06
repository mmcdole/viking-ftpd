package authorization

import (
	"testing"
	"time"

	"github.com/mmcdole/viking-ftpd/pkg/users"
)

type mockUserSource struct {
	users map[string]*users.User
}

func newMockUserSource() *mockUserSource {
	return &mockUserSource{
		users: make(map[string]*users.User),
	}
}

func (m *mockUserSource) LoadUser(username string) (*users.User, error) {
	user, ok := m.users[username]
	if !ok {
		return nil, users.ErrUserNotFound
	}
	return user, nil
}

func (m *mockUserSource) addUser(username string, level int) {
	m.users[username] = &users.User{
		Username: username,
		Level:    level,
	}
}

type mockAccessSource struct{}

func (m *mockAccessSource) LoadAccessData() (map[string]interface{}, error) {
	// Create a simple access map for testing
	return map[string]interface{}{
		"access_map": map[string]interface{}{
			"*": map[string]interface{}{
				"test": map[string]interface{}{
					".": 0, // Revoked
					"*": 0, // Revoked
				},
				"public": map[string]interface{}{
					".": 0, // Revoked
					"*": 0, // Revoked
				},
				"admin": map[string]interface{}{
					".": 0, // Revoked
					"*": 0, // Revoked
				},
			},
			"admin": map[string]interface{}{
				"?": []interface{}{"admin_group"},
				"admin": map[string]interface{}{
					".": 2, // Write
					"*": 2, // Write
				},
				"test": map[string]interface{}{
					".": 2, // Write
					"*": 2, // Write
				},
			},
			"mortal": map[string]interface{}{
				"?": []interface{}{"mortal_group"},
				"public": map[string]interface{}{
					".": 1, // Read
					"*": 1, // Read
				},
			},
			"admin_group": map[string]interface{}{
				"test": map[string]interface{}{
					".": 2, // Write
					"*": 2, // Write
				},
				"admin": map[string]interface{}{
					".": 2, // Write
					"*": 2, // Write
				},
			},
			"mortal_group": map[string]interface{}{
				"public": map[string]interface{}{
					".": 1, // Read
					"*": 1, // Read
				},
			},
		},
	}, nil
}

func TestAuthorizer_HasAccess(t *testing.T) {
	source := newMockUserSource()
	source.addUser("admin", users.ADMINISTRATOR)
	source.addUser("mortal", users.MORTAL_FIRST)

	auth := NewAuthorizer(&mockAccessSource{}, source, time.Hour)

	// Test admin access
	if !auth.HasPermission("admin", "/test", Read) {
		t.Error("Admin should have read access")
	}
	if !auth.HasPermission("admin", "/admin", Write) {
		t.Error("Admin should have write access")
	}

	// Test mortal access
	if !auth.HasPermission("mortal", "/public", Read) {
		t.Error("Mortal should have read access to public")
	}
	if auth.HasPermission("mortal", "/admin", Write) {
		t.Error("Mortal should not have write access to admin")
	}

	// Test non-existent user
	if auth.HasPermission("nonexistent", "/public", Read) {
		t.Error("Non-existent user should not have access")
	}
}

func TestAuthorizer_ResolveGroups(t *testing.T) {
	source := newMockUserSource()
	source.addUser("admin", users.ADMINISTRATOR)
	source.addUser("mortal", users.MORTAL_FIRST)

	auth := NewAuthorizer(&mockAccessSource{}, source, time.Hour)

	// Test admin groups
	adminGroups := auth.ResolveGroups("admin")
	if len(adminGroups) == 0 {
		t.Error("Admin should belong to at least one group")
	}
	if !contains(adminGroups, "admin_group") {
		t.Error("Admin should belong to admin_group")
	}

	// Test mortal groups
	mortalGroups := auth.ResolveGroups("mortal")
	if len(mortalGroups) == 0 {
		t.Error("Mortal should belong to at least one group")
	}
	if !contains(mortalGroups, "mortal_group") {
		t.Error("Mortal should belong to mortal_group")
	}

	// Test non-existent user
	nonexistentGroups := auth.ResolveGroups("nonexistent")
	if len(nonexistentGroups) != 0 {
		t.Error("Non-existent user should not belong to any groups")
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

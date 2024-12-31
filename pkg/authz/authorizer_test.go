package authz

import (
	"testing"
	"time"
)

type mockSource struct {
	data map[string]interface{}
}

func (m *mockSource) LoadRawData() (map[string]interface{}, error) {
	return m.data, nil
}

// testData returns a mock access tree that mimics a real MUD permission structure
func testData() map[string]interface{} {
	return map[string]interface{}{
		// Default access tree for all users
		"*": map[string]interface{}{
			".":          Read,    // READ on root itself
			"*":          Revoked, // REVOKED by default
			"characters": Read,    // READ on /characters
			"data":       Revoked, // REVOKED access to /data subtree
			"log":        Write,   // WRITE on logs
			"players": map[string]interface{}{
				".": Read,    // READ on /players directory itself
				"*": Revoked, // REVOKED access to all player directories
			},
		},
		// Arch users
		"knubo": map[string]interface{}{
			"?": []interface{}{"Arch_full"}, // Group membership
			"players": map[string]interface{}{
				"knubo": map[string]interface{}{
					".": Read,  // READ on directory listing
					"*": Write, // WRITE on contents
				},
			},
		},
		"frogo": map[string]interface{}{
			"?": []interface{}{"Arch_full"}, // Group membership
			"players": map[string]interface{}{
				"frogo": map[string]interface{}{
					".": Read,    // READ on directory listing
					"*": Revoked, // REVOKED on contents by default
					"com": map[string]interface{}{
						".": Write, // WRITE on /players/frogo/com directory
						"*": Write, // WRITE on contents
					},
				},
			},
		},
		// Group access trees (must start with capital letter)
		"Arch_full": map[string]interface{}{
			"players": map[string]interface{}{
				"*": GrantRead, // GRANT_READ on all player directories
			},
			"log": Write, // WRITE on logs
		},
		"Arch_docs": map[string]interface{}{
			"doc": map[string]interface{}{
				"*": GrantWrite, // GRANT_WRITE on documentation
			},
		},
		// Super admin
		"dios": map[string]interface{}{
			"*": GrantGrant, // GRANT_GRANT on everything
		},
	}
}

func TestAuthorizer(t *testing.T) {
	source := &mockSource{
		data: testData(),
	}

	auth, err := NewAuthorizer(source, time.Hour)
	if err != nil {
		t.Fatalf("creating authorizer: %v", err)
	}

	// Add debug logging for the default tree
	if tree, ok := auth.(*authorizer).trees["*"]; ok {
		t.Logf("Default tree root: DotAccess=%v, StarAccess=%v", tree.Root.DotAccess, tree.Root.StarAccess)
	} else {
		t.Log("No default tree found")
	}

	t.Run("DefaultPermissions", func(t *testing.T) {
		cases := []struct {
			name     string
			username string
			path     string
			perm     Permission
		}{
			{
				name:     "read on root",
				username: "tundra",
				path:     "/",
				perm:     Read,
			},
			{
				name:     "read on characters directory",
				username: "tundra",
				path:     "/characters",
				perm:     Read,
			},
			{
				name:     "cannot access data dir",
				username: "tundra",
				path:     "/data/notes",
				perm:     Revoked,
			},
			{
				name:     "can write to logs",
				username: "tundra",
				path:     "/log/driver",
				perm:     Write,
			},
			{
				name:     "can list players directory",
				username: "tundra",
				path:     "/players",
				perm:     Read,
			},
			{
				name:     "cannot access random player dir",
				username: "tundra",
				path:     "/players/dios/workroom.c",
				perm:     Revoked,
			},
			{
				name:     "can read open directories",
				username: "tundra",
				path:     "/players/random/open/file.txt",
				perm:     Read,
			},
		}
		runEffectivePermissionTests(t, auth, cases)
	})

	t.Run("ImplicitPlayerPermissions", func(t *testing.T) {
		cases := []struct {
			name     string
			username string
			path     string
			perm     Permission
		}{
			{
				name:     "has grant_grant on own directory",
				username: "mousepad",
				path:     "/players/mousepad",
				perm:     GrantGrant,
			},
			{
				name:     "has grant_grant on own files",
				username: "mousepad",
				path:     "/players/mousepad/test.txt",
				perm:     GrantGrant,
			},
			{
				name:     "cannot access other player directory",
				username: "mousepad",
				path:     "/players/frogo/workroom.c",
				perm:     Revoked,
			},
			{
				name:     "can read open directories",
				username: "mousepad",
				path:     "/players/frogo/open/file.txt",
				perm:     Read,
			},
		}
		runEffectivePermissionTests(t, auth, cases)
	})

	t.Run("ArchFullPermissions", func(t *testing.T) {
		cases := []struct {
			name     string
			username string
			path     string
			perm     Permission
		}{
			{
				name:     "has grant_grant on own directory",
				username: "knubo",
				path:     "/players/knubo",
				perm:     GrantGrant,
			},
			{
				name:     "can grant read on other player dirs",
				username: "knubo",
				path:     "/players/mousepad/workroom.c",
				perm:     GrantRead,
			},
			{
				name:     "can write to logs",
				username: "knubo",
				path:     "/log/driver",
				perm:     Write,
			},
		}
		runEffectivePermissionTests(t, auth, cases)
	})

	t.Run("ArchDocsPermissions", func(t *testing.T) {
		auth, err := NewAuthorizer(&mockSource{data: map[string]interface{}{
			"tundra": map[string]interface{}{
				"?": []interface{}{"Arch_docs"},
			},
			"Arch_docs": map[string]interface{}{
				"doc": map[string]interface{}{
					"*": GrantWrite,
				},
			},
		}}, time.Minute)
		if err != nil {
			t.Fatalf("creating authorizer: %v", err)
		}

		cases := []struct {
			name     string
			username string
			path     string
			perm     Permission
		}{
			{
				name:     "can grant write on docs",
				username: "tundra",
				path:     "/doc/guide.txt",
				perm:     GrantWrite,
			},
		}
		runEffectivePermissionTests(t, auth, cases)
	})

	t.Run("SuperAdminPermissions", func(t *testing.T) {
		cases := []struct {
			name     string
			username string
			path     string
			perm     Permission
		}{
			{
				name:     "has full access everywhere",
				username: "dios",
				path:     "/",
				perm:     GrantGrant,
			},
			{
				name:     "has full access to data",
				username: "dios",
				path:     "/data/notes",
				perm:     GrantGrant,
			},
		}
		runEffectivePermissionTests(t, auth, cases)
	})

	t.Run("InheritanceAndOverrides", func(t *testing.T) {
		auth, err := NewAuthorizer(&mockSource{data: map[string]interface{}{
			"*": map[string]interface{}{
				".": Read,
				"*": Revoked,
				"characters": map[string]interface{}{
					".": Read,
					"*": Revoked,
				},
				"data": map[string]interface{}{
					".": Revoked,
					"*": Revoked,
				},
				"log": map[string]interface{}{
					".": Write,
					"*": Write,
				},
				"players": map[string]interface{}{
					".": Read,
					"*": Revoked,
					"frogo": map[string]interface{}{
						".": Read,
						"*": Revoked,
					},
				},
			},
		}}, time.Minute)
		if err != nil {
			t.Fatalf("creating authorizer: %v", err)
		}

		cases := []struct {
			name     string
			username string
			path     string
			perm     Permission
		}{
			{
				name:     "can list players directory",
				username: "tundra",
				path:     "/players",
				perm:     Read,
			},
			{
				name:     "can list specific player directory",
				username: "tundra",
				path:     "/players/frogo",
				perm:     Read,
			},
			{
				name:     "cannot access player files",
				username: "tundra",
				path:     "/players/frogo/workroom.c",
				perm:     Revoked,
			},
		}
		runEffectivePermissionTests(t, auth, cases)
	})

	t.Run("GroupMembership", func(t *testing.T) {
		groups := auth.GetUserGroups("knubo")
		if len(groups) != 1 || groups[0] != "Arch_full" {
			t.Errorf("GetUserGroups(knubo) = %v, want [Arch_full]", groups)
		}

		groups = auth.GetUserGroups("mousepad")
		if len(groups) != 0 {
			t.Errorf("GetUserGroups(mousepad) = %v, want []", groups)
		}
	})
}

// runEffectivePermissionTests is a helper function to run permission test cases
func runEffectivePermissionTests(t *testing.T, auth Authorizer, cases []struct {
	name     string
	username string
	path     string
	perm     Permission
}) {
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := auth.GetEffectivePermission(c.username, c.path)
			if got != c.perm {
				t.Errorf("GetEffectivePermission(%s, %s) = %v, want %v",
					c.username, c.path, got, c.perm)
			}
		})
	}
}

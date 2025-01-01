package authorization

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
		// User with global grant permission
		"drake": map[string]interface{}{
			"*": GrantGrant, // GRANT_GRANT on everything
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
	auth, err := NewAuthorizer(&mockSource{data: testData()}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create authorizer: %v", err)
	}

	t.Run("DefaultPermissions", func(t *testing.T) {
		cases := []struct {
			name     string
			username string
			path     string
			perm     Permission
		}{
			{
				name:     "read_on_root",
				username: "tundra",
				path:     "/",
				perm:     Read,
			},
			{
				name:     "read_on_characters_directory",
				username: "tundra",
				path:     "/characters",
				perm:     Read,
			},
			{
				name:     "cannot_access_data_dir",
				username: "tundra",
				path:     "/data/notes",
				perm:     Revoked,
			},
			{
				name:     "can_write_to_logs",
				username: "tundra",
				path:     "/log/driver",
				perm:     Write,
			},
			{
				name:     "can_list_players_directory",
				username: "tundra",
				path:     "/players",
				perm:     Read,
			},
			{
				name:     "cannot_access_random_player_dir",
				username: "tundra",
				path:     "/players/dios/workroom.c",
				perm:     Revoked,
			},
			{
				name:     "can_read_open_directories",
				username: "tundra",
				path:     "/players/random/open/file.txt",
				perm:     Read,
			},
		}
		runPermTests(t, auth, cases)
	})

	t.Run("ImplicitPlayerPermissions", func(t *testing.T) {
		cases := []struct {
			name     string
			username string
			path     string
			perm     Permission
		}{
			{
				name:     "has_grant_grant_on_own_directory",
				username: "mousepad",
				path:     "/players/mousepad",
				perm:     GrantGrant,
			},
			{
				name:     "has_grant_grant_on_own_files",
				username: "mousepad",
				path:     "/players/mousepad/test.txt",
				perm:     GrantGrant,
			},
			{
				name:     "cannot_access_other_player_directory",
				username: "mousepad",
				path:     "/players/frogo/workroom.c",
				perm:     Revoked,
			},
			{
				name:     "can_read_open_directories",
				username: "mousepad",
				path:     "/players/frogo/open/file.txt",
				perm:     Read,
			},
		}
		runPermTests(t, auth, cases)
	})

	t.Run("ArchFullPermissions", func(t *testing.T) {
		cases := []struct {
			name     string
			username string
			path     string
			perm     Permission
		}{
			{
				name:     "has_grant_grant_on_own_directory",
				username: "knubo",
				path:     "/players/knubo",
				perm:     GrantGrant,
			},
			{
				name:     "can_grant_read_on_other_player_dirs",
				username: "knubo",
				path:     "/players/mousepad/workroom.c",
				perm:     GrantRead,
			},
			{
				name:     "can_write_to_logs",
				username: "knubo",
				path:     "/log/driver",
				perm:     Write,
			},
		}
		runPermTests(t, auth, cases)
	})

	t.Run("ArchDocsPermissions", func(t *testing.T) {
		// Create a new authorizer with specific test data for this case
		auth, err := NewAuthorizer(&mockSource{data: map[string]interface{}{
			"tundra": map[string]interface{}{
				"?": []interface{}{"Arch_docs"},
			},
			"Arch_docs": map[string]interface{}{
				"doc": map[string]interface{}{
					"*": GrantWrite,
				},
			},
		}}, time.Hour)
		if err != nil {
			t.Fatalf("Failed to create authorizer: %v", err)
		}

		cases := []struct {
			name     string
			username string
			path     string
			perm     Permission
		}{
			{
				name:     "can_grant_write_on_docs",
				username: "tundra",
				path:     "/doc/guide.txt",
				perm:     GrantWrite,
			},
		}
		runPermTests(t, auth, cases)
	})

	t.Run("SuperAdminPermissions", func(t *testing.T) {
		cases := []struct {
			name     string
			username string
			path     string
			perm     Permission
		}{
			{
				name:     "has_full_access_everywhere",
				username: "dios",
				path:     "/",
				perm:     GrantGrant,
			},
			{
				name:     "has_full_access_to_data",
				username: "dios",
				path:     "/data/notes",
				perm:     GrantGrant,
			},
		}
		runPermTests(t, auth, cases)
	})

	t.Run("GlobalGrantPermissions", func(t *testing.T) {
		cases := []struct {
			name     string
			username string
			path     string
			perm     Permission
		}{
			{
				name:     "drake_root_access",
				username: "drake",
				path:     "/",
				perm:     GrantGrant,
			},
			{
				name:     "drake_players_access",
				username: "drake",
				path:     "/players",
				perm:     GrantGrant,
			},
			{
				name:     "drake_other_player_access",
				username: "drake",
				path:     "/players/knubo",
				perm:     GrantGrant,
			},
			{
				name:     "drake_own_dir_access",
				username: "drake",
				path:     "/players/drake",
				perm:     GrantGrant,
			},
		}
		runPermTests(t, auth, cases)
	})

	t.Run("InheritanceAndOverrides", func(t *testing.T) {
		// Create a new authorizer with specific test data for this case
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
		}}, time.Hour)
		if err != nil {
			t.Fatalf("Failed to create authorizer: %v", err)
		}

		cases := []struct {
			name     string
			username string
			path     string
			perm     Permission
		}{
			{
				name:     "can_list_players_directory",
				username: "tundra",
				path:     "/players",
				perm:     Read,
			},
			{
				name:     "can_list_specific_player_directory",
				username: "tundra",
				path:     "/players/frogo",
				perm:     Read,
			},
			{
				name:     "cannot_access_player_files",
				username: "tundra",
				path:     "/players/frogo/workroom.c",
				perm:     Revoked,
			},
		}
		runPermTests(t, auth, cases)
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

// runPermTests is a helper function to run permission test cases
func runPermTests(t *testing.T, auth *Authorizer, cases []struct {
	name     string
	username string
	path     string
	perm     Permission
}) {
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := auth.GetEffectivePermission(tc.username, tc.path)
			if got != tc.perm {
				t.Errorf("GetEffectivePermission(%q, %q) = %v, want %v",
					tc.username, tc.path, got, tc.perm)
			}
		})
	}
}

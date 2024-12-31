package authz

import (
	"testing"
	"time"
)

type mockSource2 struct {
	data map[string]interface{}
}

func (m *mockSource2) LoadRawData() (map[string]interface{}, error) {
	return m.data, nil
}

// testData2 returns a mock access tree that covers various edge cases and permission scenarios
func testData2() map[string]interface{} {
	return map[string]interface{}{
		// Default access tree for all users
		"*": map[string]interface{}{
			".":     Read,    // Root directory itself
			"*":     Revoked, // Default for unspecified paths
			"data":  Revoked, // Explicitly revoked subtree
			"log":   Write,   // Write access to entire subtree
			"admin": GrantGrant,
			"players": map[string]interface{}{
				".": Read,    // Directory listing
				"*": Revoked, // Default for player dirs
			},
		},
		// Group access trees (capital letters)
		"Admins": map[string]interface{}{
			"admin": GrantGrant,
			"log":   GrantWrite,
		},
		"Moderators": map[string]interface{}{
			"players": map[string]interface{}{
				"*": Read,
			},
		},
		// Player access trees (lowercase letters)
		"alice": map[string]interface{}{
			"?": []interface{}{"Admins", "Moderators"}, // Multiple group membership
			"players": map[string]interface{}{
				"alice": map[string]interface{}{
					".": Read,
					"*": Write,
				},
			},
		},
		"bob": map[string]interface{}{
			"?": []interface{}{"Moderators"},
			"players": map[string]interface{}{
				"bob": map[string]interface{}{
					".":   Read,
					"*":   Revoked,
					"com": Write, // Specific directory override
				},
			},
		},
		"charlie": map[string]interface{}{
			"players": map[string]interface{}{
				"charlie": map[string]interface{}{
					".": Read,
					"*": Read,
					"private": map[string]interface{}{
						".": Write,
						"*": Revoked,
					},
				},
			},
		},
	}
}

func TestAuthorizer2(t *testing.T) {
	auth, err := NewAuthorizer(&mockSource2{data: testData2()}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create authorizer: %v", err)
	}

	// 1. Root Level Access Tests
	t.Run("root_level_access", func(t *testing.T) {
		cases := []struct {
			name     string
			username string
			path     string
			want     Permission
		}{
			{"root_dot_access", "anonymous", "/", Read},
			{"root_star_default", "anonymous", "/unknown", Revoked},
			{"root_explicit_override", "anonymous", "/data", Revoked},
		}
		runPermissionTests(t, auth, cases)
	})

	// 2. Dot vs Star Tests
	t.Run("dot_vs_star", func(t *testing.T) {
		cases := []struct {
			name     string
			username string
			path     string
			want     Permission
		}{
			{"players_dot_only", "anonymous", "/players", Read},
			{"players_star_effect", "anonymous", "/players/unknown", Revoked},
			{"nested_dot_override", "charlie", "/players/charlie", Read},
			{"nested_star_effect", "charlie", "/players/charlie/unknown", Read},
		}
		runPermissionTests(t, auth, cases)
	})

	// 3. Path Override Tests
	t.Run("path_overrides", func(t *testing.T) {
		cases := []struct {
			name     string
			username string
			path     string
			want     Permission
		}{
			{"explicit_override", "anonymous", "/log", Write},
			{"nested_override", "bob", "/players/bob/com", Write},
			{"deep_override", "charlie", "/players/charlie/private", Write},
			{"deep_override_contents", "charlie", "/players/charlie/private/file", Revoked},
		}
		runPermissionTests(t, auth, cases)
	})

	// 4. Group Access Tests
	t.Run("group_access", func(t *testing.T) {
		cases := []struct {
			name     string
			username string
			path     string
			want     Permission
		}{
			{"admin_group_access", "alice", "/admin", GrantGrant},
			{"moderator_group_access", "bob", "/players/other", Read},
			{"multiple_groups", "alice", "/log", GrantWrite},
			{"group_override_default", "bob", "/players/unknown", Read},
		}
		runPermissionTests(t, auth, cases)
	})

	// 5. Player Access Tests
	t.Run("player_access", func(t *testing.T) {
		cases := []struct {
			name     string
			username string
			path     string
			want     Permission
		}{
			{"own_directory", "alice", "/players/alice", Read},
			{"own_files", "alice", "/players/alice/file", Write},
			{"other_player_dir", "alice", "/players/bob", Read},
			{"other_player_files", "alice", "/players/bob/file", Read},
		}
		runPermissionTests(t, auth, cases)
	})

	// 6. Edge Cases
	t.Run("edge_cases", func(t *testing.T) {
		cases := []struct {
			name     string
			username string
			path     string
			want     Permission
		}{
			{"empty_path", "anonymous", "", Read},
			{"double_slash", "anonymous", "//players//bob", Revoked},
			{"dot_in_path", "bob", "/players/./bob", Read},
			{"very_deep_path", "charlie", "/players/charlie/a/b/c/d/e", Read},
		}
		runPermissionTests(t, auth, cases)
	})

	// 7. Complex Scenarios
	t.Run("complex_scenarios", func(t *testing.T) {
		cases := []struct {
			name     string
			username string
			path     string
			want     Permission
		}{
			{"mixed_permissions", "charlie", "/players/charlie/private/secret", Revoked},
			{"group_cascade", "alice", "/players/unknown/file", Read},
			{"nested_overrides", "bob", "/players/bob/com/file", Write},
			{"multi_level_inheritance", "charlie", "/players/charlie/private/dir/file", Revoked},
		}
		runPermissionTests(t, auth, cases)
	})
}

// runPermissionTests is a helper function to run permission test cases
func runPermissionTests(t *testing.T, auth Authorizer, cases []struct {
	name     string
	username string
	path     string
	want     Permission
}) {
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := auth.GetEffectivePermission(tc.username, tc.path)
			if got != tc.want {
				t.Errorf("GetEffectivePermission(%q, %q) = %v, want %v",
					tc.username, tc.path, got, tc.want)
			}
		})
	}
}

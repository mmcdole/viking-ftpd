package authorization

import (
	"testing"
	"time"
)

// mockSource provides a static data source for testing
type mockSource struct {
	data map[string]interface{}
}

func (m *mockSource) LoadRawData() (map[string]interface{}, error) {
	return m.data, nil
}

type testCase struct {
	name     string
	username string
	path     string
	want     Permission
}

// runTests is a helper function to run permission test cases
func runTests(t *testing.T, auth *Authorizer, cases []testCase) {
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

// productionTree returns a simplified version of the MUD's access tree structure
func productionTree() map[string]interface{} {
	return map[string]interface{}{
		"access_map": map[string]interface{}{
			// Default access tree for all users
			"*": map[string]interface{}{
				".":          Read,    // READ on root itself
				"*":          Revoked, // REVOKED by default
				"accounts":   Revoked,
				"characters": Revoked,
				"data":       Revoked,
				"log": map[string]interface{}{
					"*":      Read,
					"Driver": Revoked,
				},
				"players": map[string]interface{}{
					".": Read,    // READ on /players directory itself
					"*": Revoked, // REVOKED access to all player directories
				},
				"tmp": Write,
			},
			// Users with group memberships
			"user1": map[string]interface{}{},
			"user2": map[string]interface{}{},
			// Documentation archwizard
			"archwizard": map[string]interface{}{
				"?": []interface{}{"Arch_doc"}, // Group membership
				"d": map[string]interface{}{
					"MyDomain": Write, // Personal domain with write access
				},
			},
			// Domain wizard with specific area access
			"wizard": map[string]interface{}{
				"d": map[string]interface{}{
					"MyRealm": Write,
					"SharedRealm": map[string]interface{}{
						".": Write,
						"*": Read,
					},
				},
			},
			// Groups
			"Arch_doc": map[string]interface{}{
				"doc": map[string]interface{}{
					"*": GrantWrite, // Can grant write on documentation
				},
				"help": Write, // Can write help files
				"com": map[string]interface{}{
					"help": Write, // Can write command help
				},
			},
		},
	}
}

// coreTree returns a minimal tree for testing basic permission concepts
func coreTree() map[string]interface{} {
	return map[string]interface{}{
		"access_map": map[string]interface{}{
			"*": map[string]interface{}{
				".": Read,
				"*": Revoked,
				"public": map[string]interface{}{
					".": Read,
					"*": Read,
				},
				// Truly private - no access at all
				"private": map[string]interface{}{
					".": Revoked,
					"*": Revoked,
				},
				// Restricted - can see but not enter
				"restricted": map[string]interface{}{
					".": Read,
					"*": Revoked,
				},
				// Test inheritance - star permission with no dot at lower level
				"inherit": map[string]interface{}{
					"*": Write, // This should be inherited by subdirs
				},
				// Test precedence between different levels
				"mixed": map[string]interface{}{
					"*": Write,
					"sub": map[string]interface{}{
						"*": Read, // This should override parent's Write
					},
					"deep": map[string]interface{}{
						"*": GrantRead,
						"subsub": map[string]interface{}{
							"*": Write, // Test multi-level precedence
						},
					},
				},
				// Test different permission levels between dot and star
				"levels": map[string]interface{}{
					".": GrantWrite,
					"*": Write,
					"sub": map[string]interface{}{
						".": Write,
						"*": GrantRead,
					},
				},
			},
			// User-specific overrides
			"user": map[string]interface{}{
				"override": Write, // Simple override of default Revoked
				"special": map[string]interface{}{
					".": Write,
					"*": Read,
				},
			},
		},
	}
}

// groupTree returns a tree focused on testing group membership and inheritance
func groupTree() map[string]interface{} {
	return map[string]interface{}{
		"access_map": map[string]interface{}{
			"*": map[string]interface{}{
				".": Read,
				"*": Revoked,
			},
			"Group1": map[string]interface{}{
				"area1": Write,
			},
			"Group2": map[string]interface{}{
				"area2": GrantRead,
			},
			"user1": map[string]interface{}{
				"?": []interface{}{"Group1", "Group2"}, // Member of both groups
			},
			"user2": map[string]interface{}{
				"?": []interface{}{"Group2"}, // Member of Group2 only
			},
		},
	}
}

func TestProductionExample(t *testing.T) {
	auth, err := NewAuthorizer(&mockSource{data: productionTree()}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create authorizer: %v", err)
	}

	t.Run("DefaultAccess", func(t *testing.T) {
		cases := []testCase{
			{"root_access", "anonymous", "/", Read},                    // Everyone can read root directory
			{"accounts_denied", "anonymous", "/accounts", Revoked},     // Accounts dir is private
			{"characters_denied", "anonymous", "/characters", Revoked}, // Character data is private
			{"data_denied", "anonymous", "/data/test", Revoked},        // Data dir is private
			{"log_readable", "anonymous", "/log/test.log", Read},       // Logs are readable except Driver
			{"log_driver_denied", "anonymous", "/log/Driver", Revoked}, // Driver logs are private
			{"tmp_writable", "anonymous", "/tmp", Write},               // Temp dir is writable by all
		}
		runTests(t, auth, cases)
	})

	t.Run("ImplicitPermissions", func(t *testing.T) {
		cases := []testCase{
			// Player's own directory permissions
			{"own_dir_root", "wizard", "/players/wizard", GrantGrant},                         // Full access to own root
			{"own_dir_file", "wizard", "/players/wizard/file.txt", GrantGrant},                // Full access to own files
			{"own_dir_subdir", "wizard", "/players/wizard/subdir", GrantGrant},                // Full access to subdirs
			{"own_dir_deep_file", "wizard", "/players/wizard/deep/path/file.txt", GrantGrant}, // Full access at any depth
			{"own_open_dir", "wizard", "/players/wizard/open", GrantGrant},                    // Full access to own open dir

			// Other players' open directory permissions
			{"other_open_dir", "wizard", "/players/other/open", Read},            // Can read others' open dirs
			{"archwizard_open_dir", "archwizard", "/players/someone/open", Read}, // Even archwizards only get read
			{"anonymous_open_dir", "anonymous", "/players/anyone/open", Read},    // Anonymous can read open dirs

			// Verify open dir only works at correct level
			{"open_dir_file_denied", "anonymous", "/players/anyone/open/file.txt", Revoked}, // Can't read files in open
			{"open_string_in_path", "anonymous", "/players/anyone/not_open", Revoked},       // 'open' must be exact dir name
			{"deep_open_denied", "anonymous", "/players/anyone/subdir/open", Revoked},       // 'open' must be at right level
		}
		runTests(t, auth, cases)
	})

	t.Run("GroupAndDomainAccess", func(t *testing.T) {
		cases := []testCase{
			// Arch_doc group permissions
			{"doc_grant_write", "archwizard", "/doc/manual.txt", GrantWrite},     // Can grant write on documentation
			{"help_write", "archwizard", "/help/newbie.txt", Write},              // Can write help files
			{"command_help_write", "archwizard", "/com/help/command.txt", Write}, // Can write command help files

			// Personal domain access
			{"own_domain_write", "archwizard", "/d/MyDomain/area.c", Write}, // Can write in personal domain

			// Domain wizard permissions
			{"realm_write", "wizard", "/d/MyRealm/room.c", Write},           // Full write in own realm
			{"shared_realm_root", "wizard", "/d/SharedRealm", Write},        // Write at shared realm root
			{"shared_realm_files", "wizard", "/d/SharedRealm/room.c", Read}, // Only read in shared files

			// Interaction between group and implicit
			{"doc_in_home", "archwizard", "/players/archwizard/doc.txt", GrantGrant}, // Home dir trumps group perms
			{"open_with_domain", "wizard", "/players/wizard/open", GrantGrant},       // Home dir trumps domain perms
		}
		runTests(t, auth, cases)
	})
}

func TestCorePermissions(t *testing.T) {
	auth, err := NewAuthorizer(&mockSource{data: coreTree()}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create authorizer: %v", err)
	}

	t.Run("DotVsStar", func(t *testing.T) {
		cases := []testCase{
			{"root_dot", "anonymous", "/", Read},               // Root dir is readable
			{"root_star", "anonymous", "/unknown", Revoked},    // Unknown paths are denied
			{"public_dot", "anonymous", "/public", Read},       // Can read public dir
			{"public_star", "anonymous", "/public/file", Read}, // Can read files in public

			// Private directory - no access at all
			{"private_dot", "anonymous", "/private", Revoked},       // Can't even see private dir
			{"private_star", "anonymous", "/private/file", Revoked}, // Can't access private files

			// Restricted - can see but not enter
			{"restricted_dot", "anonymous", "/restricted", Read},          // Can see restricted dir
			{"restricted_star", "anonymous", "/restricted/file", Revoked}, // But can't access contents
		}
		runTests(t, auth, cases)
	})

	t.Run("PermissionInheritance", func(t *testing.T) {
		cases := []testCase{
			{"inherit_star", "anonymous", "/inherit/file", Write},            // Star permission applies to files
			{"inherit_subdir", "anonymous", "/inherit/sub", Write},           // Star permission applies to subdirs
			{"inherit_subdir_file", "anonymous", "/inherit/sub/file", Write}, // Star permission applies at any depth
		}
		runTests(t, auth, cases)
	})

	t.Run("PermissionPrecedence", func(t *testing.T) {
		cases := []testCase{
			{"mixed_root_file", "anonymous", "/mixed/file", Write},           // Gets parent's * permission
			{"mixed_sub_file", "anonymous", "/mixed/sub/file", Read},         // Subdir * overrides parent
			{"mixed_deep_file", "anonymous", "/mixed/deep/file", GrantRead},  // Deeper permission wins
			{"mixed_deepest", "anonymous", "/mixed/deep/subsub/file", Write}, // Deepest level always wins
		}
		runTests(t, auth, cases)
	})

	t.Run("MixedPermissionLevels", func(t *testing.T) {
		cases := []testCase{
			{"levels_dot", "anonymous", "/levels", GrantWrite},              // Dot can have higher permission
			{"levels_file", "anonymous", "/levels/file", Write},             // Star has its own permission
			{"levels_sub_dot", "anonymous", "/levels/sub", Write},           // Subdir dot permission
			{"levels_sub_file", "anonymous", "/levels/sub/file", GrantRead}, // Subdir star can be higher
		}
		runTests(t, auth, cases)
	})

	t.Run("UserOverrides", func(t *testing.T) {
		cases := []testCase{
			{"simple_override", "user", "/override", Write}, // User overrides default Revoked
			{"special_dot", "user", "/special", Write},      // User-specific dot permission
			{"special_star", "user", "/special/file", Read}, // User-specific star permission
		}
		runTests(t, auth, cases)
	})
}

func TestGroupPermissions(t *testing.T) {
	auth, err := NewAuthorizer(&mockSource{data: groupTree()}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create authorizer: %v", err)
	}

	t.Run("SingleGroup", func(t *testing.T) {
		cases := []testCase{
			{"area2_access", "user2", "/area2", GrantRead}, // Member of Group2 gets its permissions
			{"area1_denied", "user2", "/area1", Revoked},   // No access to Group1's area
		}
		runTests(t, auth, cases)
	})

	t.Run("MultipleGroups", func(t *testing.T) {
		cases := []testCase{
			{"area1_access", "user1", "/area1", Write},     // Gets Write from Group1
			{"area2_access", "user1", "/area2", GrantRead}, // Gets GrantRead from Group2
		}
		runTests(t, auth, cases)
	})
}

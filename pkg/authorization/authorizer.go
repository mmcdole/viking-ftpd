package authorization

import (
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/viking-ftpd/pkg/playerdata"
)

// AccessTree represents a node in the access permission tree
type AccessTree struct {
	Root   *AccessNode
	Groups []string
}

// AccessNode represents a node in the access tree
type AccessNode struct {
	DotAccess  Permission
	StarAccess Permission
	Children   map[string]*AccessNode
}

// Authorizer handles access control and permissions with caching
type Authorizer struct {
	source        AccessSource
	characterData playerdata.Source
	cacheDuration time.Duration

	mu          sync.RWMutex
	trees       map[string]*AccessTree
	lastRefresh time.Time
	charSource  playerdata.Source
}

// NewAuthorizer creates a new Authorizer instance
func NewAuthorizer(source AccessSource, characterData playerdata.Source, cacheDuration time.Duration) (*Authorizer, error) {
	a := &Authorizer{
		source:        source,
		characterData: characterData,
		cacheDuration: cacheDuration,
		charSource:    characterData,
		trees:         make(map[string]*AccessTree),
	}

	// Initial load
	if err := a.refreshCache(); err != nil {
		return nil, err
	}

	return a, nil
}

// refreshCache loads fresh data from the source
func (a *Authorizer) refreshCache() error {
	rawData, err := a.source.LoadRawData()
	if err != nil {
		return fmt.Errorf("loading raw data: %w", err)
	}

	trees, err := ConvertToAccessTrees(rawData)
	if err != nil {
		return fmt.Errorf("converting access trees: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.trees = trees
	a.lastRefresh = time.Now()
	return nil
}

// ensureFreshCache checks if cache needs refresh
func (a *Authorizer) ensureFreshCache() error {
	a.mu.RLock()
	needsRefresh := time.Since(a.lastRefresh) >= a.cacheDuration
	a.mu.RUnlock()

	if needsRefresh {
		return a.refreshCache()
	}
	return nil
}

// getImplicitPermission returns any implicit permissions for a path and user
func (a *Authorizer) getImplicitPermission(username string, parts []string) (Permission, bool) {
	if len(parts) >= 2 && parts[0] == "players" {
		if parts[1] == username {
			return GrantGrant, true // Users always have GRANT_GRANT on their own directory
		}
		// Check for open directory at exactly level 3
		if len(parts) >= 3 && parts[2] == "open" && len(parts) == 3 {
			return Read, true // Everyone can read open directories at level 3
		}
	}
	return Revoked, false
}

// getImplicitGroups returns implicit groups based on character level
func (a *Authorizer) getImplicitGroups(username string) []string {
	if err := a.ensureFreshCache(); err != nil {
		return nil
	}

	if a.charSource == nil {
		return nil
	}

	char, err := a.charSource.LoadCharacter(username)
	if err != nil {
		return nil
	}

	var groups []string

	// Check if the groups exist in the access map before adding them
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Arch_full for archwizards and above
	if _, ok := a.trees[GroupArchFull]; ok && char.Level >= playerdata.ARCHWIZARD {
		groups = append(groups, GroupArchFull)
	} else if _, ok := a.trees[GroupArchJunior]; ok && char.Level >= playerdata.JUNIOR_ARCH && char.Level != playerdata.ELDER {
		// Arch_junior for junior arches (except elders)
		groups = append(groups, GroupArchJunior)
	}

	return groups
}

// HasPermission implements Authorizer
func (a *Authorizer) HasPermission(username string, filepath string, requiredPerm Permission) bool {
	if err := a.ensureFreshCache(); err != nil {
		return false
	}
	effectivePerm := a.GetEffectivePermission(username, filepath)
	return effectivePerm >= requiredPerm
}

// GetEffectivePermission implements Authorizer
func (a *Authorizer) GetEffectivePermission(username string, filepath string) Permission {
	if err := a.ensureFreshCache(); err != nil {
		return Revoked
	}

	// Clean the path and split into parts
	parts := strings.Split(path.Clean(filepath), "/")
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}
	// Handle root path specifically
	if len(parts) == 1 && parts[0] == "" {
		parts = []string{} // Empty array for root path
	}

	// Check implicit permissions first
	if implicitPerm, ok := a.getImplicitPermission(username, parts); ok {
		return implicitPerm
	}

	// Check user's direct permissions
	if tree, ok := a.trees[username]; ok {
		perm := a.checkNodePermission(tree.Root, parts)
		if perm != Revoked {
			return perm
		}
	}

	// Check all group permissions (both explicit and implicit)
	for _, group := range a.GetGroups(username) {
		if tree, ok := a.trees[group]; ok {
			perm := a.checkNodePermission(tree.Root, parts)
			if perm != Revoked {
				return perm
			}
		}
	}

	// Finally check default permissions
	if tree, ok := a.trees["*"]; ok {
		return a.checkNodePermission(tree.Root, parts)
	}

	return Revoked
}

// GetUserGroups implements Authorizer
func (a *Authorizer) GetUserGroups(username string) []string {
	if err := a.ensureFreshCache(); err != nil {
		return nil
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	// Get user's tree
	tree, ok := a.trees[username]
	if !ok || tree == nil {
		return nil
	}

	// Groups are stored in the tree itself
	return tree.Groups
}

// GetGroups returns all groups that a user belongs to, including both
// explicit groups from the access tree and implicit groups based on character level.
func (a *Authorizer) GetGroups(username string) []string {
	// Get implicit groups first
	groups := make([]string, 0)
	implicitGroups := a.getImplicitGroups(username)
	if implicitGroups != nil {
		groups = append(groups, implicitGroups...)
	}

	// Get user groups from access tree
	userGroups := make([]string, 0)
	if userNode, ok := a.trees[username]; ok {
		if userNode != nil {
			userGroups = userNode.Groups
		}
	}

	// Combine implicit and explicit groups
	if userGroups != nil {
		groups = append(groups, userGroups...)
	}

	return groups
}

// checkNodePermission recursively checks permissions in a node
func (a *Authorizer) checkNodePermission(node *AccessNode, pathParts []string) Permission {
	if node == nil {
		return Revoked
	}

	// At the target node (final node)
	if len(pathParts) == 0 {
		// At final node, dot access overrides star access
		if node.DotAccess != Revoked {
			return node.DotAccess
		}
		// No dot access, use star access
		return node.StarAccess
	}

	part := pathParts[0]
	rest := pathParts[1:]

	// Check for exact match in children
	if child, ok := node.Children[part]; ok {
		// Recursively check child permissions
		childPerm := a.checkNodePermission(child, rest)
		// If child returns Revoked, that's final - don't fall back to star access
		return childPerm
	}

	// No matching child, use star access
	return node.StarAccess
}

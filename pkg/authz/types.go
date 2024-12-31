package authz

import "time"

// Permission represents an access level in the system
type Permission int

const (
	Revoked     Permission = -1
	Read        Permission = 1
	GrantRead   Permission = 2
	Write       Permission = 3
	GrantWrite  Permission = 4
	GrantGrant  Permission = 5
)

// AccessNode represents a node in the access tree
type AccessNode struct {
	// Direct access level for this node
	DotAccess   Permission
	// Default access level for children
	StarAccess  Permission
	// Named child nodes
	Children    map[string]*AccessNode
}

// AccessTree represents the complete access hierarchy
type AccessTree struct {
	// Root node of the tree
	Root   *AccessNode
	// Groups this user belongs to (if this is a user tree)
	Groups []string
}

// AccessSource represents a source of raw access data
type AccessSource interface {
	// LoadRawData loads the raw map structure that will be converted to AccessTrees
	LoadRawData() (map[string]interface{}, error)
}

// Authorizer handles access control and permissions
type Authorizer interface {
	// HasPermission checks if a user has the required permission for a path
	HasPermission(username string, path string, requiredPerm Permission) bool

	// GetEffectivePermission returns the effective permission a user has on a path
	GetEffectivePermission(username string, path string) Permission

	// GetUserGroups returns the list of groups a user belongs to
	GetUserGroups(username string) []string
}

// CachingAuthorizer extends the base Authorizer interface with cache management
type CachingAuthorizer interface {
	Authorizer
	// RefreshCache forces a reload of the access tree from the source
	RefreshCache() error
}

// AuthorizerConfig holds the configuration for creating a new Authorizer
type AuthorizerConfig struct {
	// Source provides the access tree data
	Source AccessSource

	// DefaultPermission is used when no matching rule is found
	DefaultPermission Permission

	// CacheDuration specifies how long to cache the access tree
	// If zero, caching is disabled and every check loads fresh data
	CacheDuration time.Duration
}

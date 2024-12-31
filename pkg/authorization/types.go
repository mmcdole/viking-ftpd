package authorization

import "time"

// Permission represents an access level in the system
type Permission int

const (
	Revoked    Permission = -1
	Read       Permission = 1
	GrantRead  Permission = 2
	Write      Permission = 3
	GrantWrite Permission = 4
	GrantGrant Permission = 5
)

// CanRead returns true if the permission allows reading
func (p Permission) CanRead() bool {
	return p >= Read
}

// CanWrite returns true if the permission allows writing
func (p Permission) CanWrite() bool {
	return p >= Write
}

// CanGrant returns true if the permission allows granting permissions
func (p Permission) CanGrant() bool {
	return p >= GrantGrant
}

// AccessNode represents a node in the access tree
type AccessNode struct {
	// Direct access level for this node
	DotAccess Permission
	// Default access level for children
	StarAccess Permission
	// Named child nodes
	Children map[string]*AccessNode
}

// AccessTree represents the complete access hierarchy
type AccessTree struct {
	// Root node of the tree
	Root *AccessNode
	// Groups this user belongs to (if this is a user tree)
	Groups []string
}

// AccessSource represents a source of raw access data
type AccessSource interface {
	// LoadRawData loads the raw map structure that will be converted to AccessTrees
	LoadRawData() (map[string]interface{}, error)
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

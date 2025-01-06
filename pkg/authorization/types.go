package authorization

// AccessSource provides access to the raw access tree data
type AccessSource interface {
	LoadAccessData() (map[string]interface{}, error)
}

// Permission represents the level of access granted
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

// Group constants
const (
	GroupArchFull   = "Arch_full"
	GroupArchJunior = "Arch_junior"
	GroupArchDocs   = "Arch_docs"
	GroupArchQC     = "Arch_qc"
	GroupArchLaw    = "Arch_law"
	GroupArchWeb    = "Arch_web"
)

// AuthorizerConfig holds the configuration for creating a new Authorizer
type AuthorizerConfig struct {
	// DefaultPermission is used when no matching rule is found
	DefaultPermission Permission
}

package authorization

// AccessSource provides access to the raw access tree data
type AccessSource interface {
	LoadRawData() (map[string]interface{}, error)
}

// Permission represents the level of access granted
type Permission int

const (
	Revoked Permission = iota
	Read
	Write
	GrantRead
	GrantWrite
	GrantGrant
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

// CharacterDataSource represents a source of character level data
type CharacterDataSource interface {
	// GetCharacterLevel returns the level for a given character
	GetCharacterLevel(username string) (int, error)
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
	// CharacterData provides the character level data
	CharacterData CharacterDataSource

	// DefaultPermission is used when no matching rule is found
	DefaultPermission Permission
}

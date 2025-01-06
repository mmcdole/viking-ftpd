package users

// User represents a user in the system
type User struct {
	Username     string
	PasswordHash string
	Level        int
}

// Source represents a source of user data
type Source interface {
	// LoadUser loads user data for a given username
	LoadUser(username string) (*User, error)
}

// Constants for user levels
const (
	MORTAL_FIRST  = 1
	NEWBIE_END    = 15
	MORTAL_LAST   = 19
	ETERNAL_FIRST = 20
	ETERNAL_LAST  = 29
	STUDENT       = 30
	WIZARD        = 31
	PROCTOR       = 32
	U_LORD        = 33
	CREATOR       = 35
	ARCHITECT     = 36
	LORD          = 37
	VISITING_ARCH = 39
	JUNIOR_ARCH   = 40
	ELDER         = 42
	ARCHWIZARD    = 45
	ADMINISTRATOR = 50
)

package playerdata

import "errors"

var (
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidHash  = errors.New("invalid hash in character file")
)

// Level constants from MUD
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

// Character represents a MUD character/player
type Character struct {
	Username     string
	PasswordHash string
	Level        int
	// Add other fields as needed
}

// Source represents a source of character data
type Source interface {
	// LoadCharacter loads character data for a given username
	// Returns ErrUserNotFound if the character doesn't exist
	LoadCharacter(username string) (*Character, error)
}

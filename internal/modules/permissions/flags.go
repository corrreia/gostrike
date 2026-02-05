// Package permissions provides the permissions system for GoStrike.
// This file defines admin flags and their string representations.
package permissions

// AdminFlag represents a single permission flag
type AdminFlag uint64

// Admin flags - each represents a specific permission
const (
	// No permissions
	FlagNone AdminFlag = 0

	// Basic admin flags
	FlagReservation AdminFlag = 1 << iota // Slot reservation
	FlagKick                              // Kick players
	FlagBan                               // Ban players
	FlagUnban                             // Unban players
	FlagSlay                              // Slay/kill players
	FlagChangelevel                       // Change map
	FlagCvar                              // Change cvars
	FlagConfig                            // Execute config files
	FlagChat                              // Admin chat commands
	FlagVote                              // Start/cancel votes
	FlagPassword                          // Change server password
	FlagRcon                              // Remote console access
	FlagCheats                            // Enable cheats (sv_cheats)
	FlagCustom1                           // Custom flag 1
	FlagCustom2                           // Custom flag 2
	FlagCustom3                           // Custom flag 3
	FlagCustom4                           // Custom flag 4
	FlagCustom5                           // Custom flag 5
	FlagCustom6                           // Custom flag 6

	// Special flags
	FlagRoot AdminFlag = 1 << 63 // Full access to everything

	// Common flag combinations
	FlagGeneric = FlagKick | FlagBan | FlagSlay | FlagChat
	FlagAdmin   = FlagGeneric | FlagChangelevel | FlagCvar | FlagVote
	FlagFull    = FlagAdmin | FlagConfig | FlagPassword | FlagRcon
)

// flagNames maps flags to their string names
var flagNames = map[AdminFlag]string{
	FlagReservation: "reservation",
	FlagKick:        "kick",
	FlagBan:         "ban",
	FlagUnban:       "unban",
	FlagSlay:        "slay",
	FlagChangelevel: "changelevel",
	FlagCvar:        "cvar",
	FlagConfig:      "config",
	FlagChat:        "chat",
	FlagVote:        "vote",
	FlagPassword:    "password",
	FlagRcon:        "rcon",
	FlagCheats:      "cheats",
	FlagCustom1:     "custom1",
	FlagCustom2:     "custom2",
	FlagCustom3:     "custom3",
	FlagCustom4:     "custom4",
	FlagCustom5:     "custom5",
	FlagCustom6:     "custom6",
	FlagRoot:        "root",
}

// letterFlags maps single letters to flags (SourceMod style)
var letterFlags = map[rune]AdminFlag{
	'a': FlagReservation,
	'b': FlagGeneric,
	'c': FlagKick,
	'd': FlagBan,
	'e': FlagUnban,
	'f': FlagSlay,
	'g': FlagChangelevel,
	'h': FlagCvar,
	'i': FlagConfig,
	'j': FlagChat,
	'k': FlagVote,
	'l': FlagPassword,
	'm': FlagRcon,
	'n': FlagCheats,
	'o': FlagCustom1,
	'p': FlagCustom2,
	'q': FlagCustom3,
	'r': FlagCustom4,
	's': FlagCustom5,
	't': FlagCustom6,
	'z': FlagRoot,
}

// String returns the name of a single flag
func (f AdminFlag) String() string {
	if f == FlagNone {
		return "none"
	}
	if name, ok := flagNames[f]; ok {
		return name
	}
	return "unknown"
}

// Has checks if the flag set contains a specific flag
func (f AdminFlag) Has(flag AdminFlag) bool {
	if f&FlagRoot != 0 {
		return true // Root has all permissions
	}
	return f&flag == flag
}

// HasAny checks if the flag set contains any of the specified flags
func (f AdminFlag) HasAny(flags AdminFlag) bool {
	if f&FlagRoot != 0 {
		return true
	}
	return f&flags != 0
}

// Add adds flags to the set
func (f AdminFlag) Add(flags AdminFlag) AdminFlag {
	return f | flags
}

// Remove removes flags from the set
func (f AdminFlag) Remove(flags AdminFlag) AdminFlag {
	return f &^ flags
}

// ParseFlags parses a flag string (letter format or comma-separated names)
func ParseFlags(s string) AdminFlag {
	var result AdminFlag

	// Check for letter format first (e.g., "abcdef")
	isLetterFormat := true
	for _, c := range s {
		if c != ',' && (c < 'a' || c > 'z') {
			isLetterFormat = false
			break
		}
	}

	if isLetterFormat && len(s) > 0 && s[0] != ',' {
		for _, c := range s {
			if flag, ok := letterFlags[c]; ok {
				result |= flag
			}
		}
		return result
	}

	// Otherwise, try comma-separated names
	// Simple split by comma
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if i > start {
				name := s[start:i]
				// Trim spaces
				for len(name) > 0 && name[0] == ' ' {
					name = name[1:]
				}
				for len(name) > 0 && name[len(name)-1] == ' ' {
					name = name[:len(name)-1]
				}
				if flag := FlagByName(name); flag != FlagNone {
					result |= flag
				}
			}
			start = i + 1
		}
	}

	return result
}

// FlagByName returns a flag by its name
func FlagByName(name string) AdminFlag {
	for flag, n := range flagNames {
		if n == name {
			return flag
		}
	}
	return FlagNone
}

// ToLetters converts flags to letter format
func (f AdminFlag) ToLetters() string {
	var result []rune
	for letter, flag := range letterFlags {
		if f.Has(flag) {
			result = append(result, letter)
		}
	}
	return string(result)
}

// ToNames converts flags to comma-separated names
func (f AdminFlag) ToNames() string {
	var names []string
	for flag, name := range flagNames {
		if f.Has(flag) {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "none"
	}
	result := names[0]
	for i := 1; i < len(names); i++ {
		result += ", " + names[i]
	}
	return result
}

package permissions

import "fmt"

// ParseSteamID parses various SteamID formats.
// Supports: STEAM_X:Y:Z, [U:1:123456], 76561198012345678
func ParseSteamID(s string) (uint64, error) {
	// Check for SteamID64 format (17-digit number starting with 765)
	if len(s) == 17 && s[0] == '7' && s[1] == '6' && s[2] == '5' {
		var id uint64
		for _, c := range s {
			if c < '0' || c > '9' {
				return 0, fmt.Errorf("invalid steamid64: %s", s)
			}
			id = id*10 + uint64(c-'0')
		}
		return id, nil
	}

	// Check for STEAM_X:Y:Z format
	if len(s) > 8 && s[0:6] == "STEAM_" {
		var x, y, z uint64
		_, err := fmt.Sscanf(s, "STEAM_%d:%d:%d", &x, &y, &z)
		if err != nil {
			return 0, fmt.Errorf("invalid steam_id format: %s", s)
		}
		return 76561197960265728 + z*2 + y, nil
	}

	// Check for [U:1:Z] format
	if len(s) > 5 && s[0] == '[' && s[1] == 'U' && s[2] == ':' {
		var universe, z uint64
		_, err := fmt.Sscanf(s, "[U:%d:%d]", &universe, &z)
		if err != nil {
			return 0, fmt.Errorf("invalid steam3id format: %s", s)
		}
		return 76561197960265728 + z, nil
	}

	// Try parsing as raw number
	var id uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid steamid: %s", s)
		}
		id = id*10 + uint64(c-'0')
	}
	return id, nil
}

// FormatSteamID64 formats a SteamID64 to string.
func FormatSteamID64(id uint64) string {
	return fmt.Sprintf("%d", id)
}

// FormatSteamID2 formats a SteamID64 to STEAM_X:Y:Z format.
func FormatSteamID2(id uint64) string {
	if id < 76561197960265728 {
		return fmt.Sprintf("STEAM_0:0:%d", id)
	}
	w := id - 76561197960265728
	y := w % 2
	z := w / 2
	return fmt.Sprintf("STEAM_0:%d:%d", y, z)
}

// FormatSteamID3 formats a SteamID64 to [U:1:Z] format.
func FormatSteamID3(id uint64) string {
	if id < 76561197960265728 {
		return fmt.Sprintf("[U:1:%d]", id)
	}
	z := id - 76561197960265728
	return fmt.Sprintf("[U:1:%d]", z)
}

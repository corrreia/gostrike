// Package gostrike provides the public SDK for GoStrike plugins.
package gostrike

// ChatColor represents a CS2 chat color code.
// These work with HUD_PRINTTALK (chat) messages only.
// Use them as string prefixes: ColorGreen + "Hello" or via Colorize().
type ChatColor string

const (
	ColorDefault   ChatColor = "\x01" // White/default
	ColorDarkRed   ChatColor = "\x02" // Dark red
	ColorTeam      ChatColor = "\x03" // Team color (CT=blue, T=gold)
	ColorGreen     ChatColor = "\x04" // Green
	ColorOlive     ChatColor = "\x05" // Olive/dark green
	ColorLime      ChatColor = "\x06" // Lime/bright green
	ColorGold      ChatColor = "\x09" // Gold
	ColorGrey      ChatColor = "\x0A" // Grey
	ColorLightBlue ChatColor = "\x0B" // Light blue
	ColorBlue      ChatColor = "\x0C" // Blue
	ColorPurple    ChatColor = "\x0D" // Purple
	ColorRed       ChatColor = "\x0E" // Red
	ColorOrange    ChatColor = "\x0F" // Orange
	ColorWhite     ChatColor = "\x10" // White
)

// Colorize wraps text with a color prefix and resets to default after.
func Colorize(color ChatColor, text string) string {
	return string(color) + text + string(ColorDefault)
}

// Package gostrike provides the public SDK for GoStrike plugins.
// This file provides target pattern resolution (e.g., @all, @alive, @ct, #userid).
package gostrike

import (
	"strings"
)

// ResolveTarget resolves a target pattern string to matching players.
// Supported patterns:
//   - @all      - All players
//   - @alive    - All alive players
//   - @dead     - All dead players
//   - @ct       - All Counter-Terrorists
//   - @t        - All Terrorists
//   - @spec     - All Spectators
//   - @me       - The caller
//   - @!me      - Everyone except the caller
//   - @random   - One random alive player
//   - @bot      - All bots
//   - #<slot>   - Player by slot number (e.g., #3)
//   - <name>    - Partial name match (case-insensitive)
func ResolveTarget(caller *Player, pattern string) []*Player {
	all := GetServer().GetPlayers()
	if len(all) == 0 {
		return nil
	}

	switch strings.ToLower(pattern) {
	case "@all":
		return all

	case "@alive":
		return filterPlayers(all, func(p *Player) bool { return p.IsAlive })

	case "@dead":
		return filterPlayers(all, func(p *Player) bool { return !p.IsAlive })

	case "@ct":
		return filterPlayers(all, func(p *Player) bool { return p.Team == TeamCT })

	case "@t":
		return filterPlayers(all, func(p *Player) bool { return p.Team == TeamT })

	case "@spec":
		return filterPlayers(all, func(p *Player) bool { return p.Team == TeamSpectator })

	case "@me":
		if caller != nil {
			return []*Player{caller}
		}
		return nil

	case "@!me":
		if caller == nil {
			return all
		}
		return filterPlayers(all, func(p *Player) bool { return p.Slot != caller.Slot })

	case "@random":
		alive := filterPlayers(all, func(p *Player) bool { return p.IsAlive })
		if len(alive) > 0 {
			return alive[:1]
		}
		return nil

	case "@bot":
		return filterPlayers(all, func(p *Player) bool { return p.IsBot })
	}

	// Check for #<slot> pattern
	if strings.HasPrefix(pattern, "#") {
		var slot int
		if _, err := parseSlot(pattern[1:], &slot); err == nil {
			for _, p := range all {
				if p.Slot == slot {
					return []*Player{p}
				}
			}
		}
		return nil
	}

	// Partial name match (case-insensitive)
	lower := strings.ToLower(pattern)
	return filterPlayers(all, func(p *Player) bool {
		return strings.Contains(strings.ToLower(p.Name), lower)
	})
}

// filterPlayers returns players matching the predicate
func filterPlayers(players []*Player, pred func(*Player) bool) []*Player {
	var result []*Player
	for _, p := range players {
		if pred(p) {
			result = append(result, p)
		}
	}
	return result
}

// parseSlot parses a slot number string
func parseSlot(s string, slot *int) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	*slot = n
	return 1, nil
}

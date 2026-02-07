// Package gostrike provides the public SDK for GoStrike plugins.
// This file provides a chat-based menu system.
package gostrike

import (
	"fmt"
	"sync"
	"time"
)

// MenuCallback is called when a player selects a menu item
type MenuCallback func(player *Player)

// MenuItem represents a single menu option
type MenuItem struct {
	Label    string
	Callback MenuCallback
}

// Menu represents a chat-based selection menu
type Menu struct {
	Title   string
	Items   []MenuItem
	Timeout time.Duration
}

// NewMenu creates a new menu with the given title
func NewMenu(title string) *Menu {
	return &Menu{
		Title:   title,
		Timeout: 20 * time.Second,
	}
}

// AddItem adds an option to the menu
func (m *Menu) AddItem(label string, callback MenuCallback) *Menu {
	m.Items = append(m.Items, MenuItem{Label: label, Callback: callback})
	return m
}

// SetTimeout sets the menu timeout duration
func (m *Menu) SetTimeout(d time.Duration) *Menu {
	m.Timeout = d
	return m
}

// Display shows the menu to a player and listens for their selection
func (m *Menu) Display(player *Player, timeout ...time.Duration) {
	if player == nil || len(m.Items) == 0 {
		return
	}

	t := m.Timeout
	if len(timeout) > 0 {
		t = timeout[0]
	}

	// Show menu to player
	player.PrintToChat("=== %s ===", m.Title)
	for i, item := range m.Items {
		player.PrintToChat(" %d. %s", i+1, item.Label)
	}
	player.PrintToChat("Type a number in chat to select")

	// Register this menu for the player
	registerActiveMenu(player.Slot, m)

	// Start timeout
	go func() {
		time.Sleep(t)
		removeActiveMenu(player.Slot)
	}()
}

// ============================================================
// Active Menu Tracking
// ============================================================

var (
	activeMenus   = make(map[int]*Menu)
	activeMenusMu sync.RWMutex
)

func registerActiveMenu(slot int, menu *Menu) {
	activeMenusMu.Lock()
	defer activeMenusMu.Unlock()
	activeMenus[slot] = menu
}

func removeActiveMenu(slot int) {
	activeMenusMu.Lock()
	defer activeMenusMu.Unlock()
	delete(activeMenus, slot)
}

// HandleMenuSelection processes a chat message as a potential menu selection.
// Returns true if the message was consumed by a menu.
func HandleMenuSelection(slot int, message string) bool {
	activeMenusMu.RLock()
	menu, exists := activeMenus[slot]
	activeMenusMu.RUnlock()

	if !exists || menu == nil {
		return false
	}

	// Parse selection number
	var selection int
	_, err := fmt.Sscanf(message, "%d", &selection)
	if err != nil || selection < 1 || selection > len(menu.Items) {
		return false
	}

	// Execute callback
	item := menu.Items[selection-1]
	removeActiveMenu(slot)

	// Get the player for the callback
	player := GetServer().GetPlayerBySlot(slot)
	if player != nil && item.Callback != nil {
		item.Callback(player)
	}

	return true
}

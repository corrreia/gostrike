// Package gostrike provides the public SDK for GoStrike plugins.
// This file provides a simple localization/translation system.
package gostrike

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Localizer handles translation of messages for plugins
type Localizer struct {
	pluginName string
	mu         sync.RWMutex
	langs      map[string]map[string]string // locale -> key -> translation
	fallback   string                       // fallback locale
}

// NewLocalizer creates a new Localizer for a plugin
func NewLocalizer(pluginName string) *Localizer {
	return &Localizer{
		pluginName: pluginName,
		langs:      make(map[string]map[string]string),
		fallback:   "en",
	}
}

// SetFallback sets the fallback locale (default: "en")
func (l *Localizer) SetFallback(locale string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fallback = locale
}

// LoadLangDir loads all .json language files from a directory.
// File names should be locale codes (e.g., en.json, pt.json, de.json).
func (l *Localizer) LoadLangDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read lang dir %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		locale := strings.TrimSuffix(entry.Name(), ".json")
		path := filepath.Join(dir, entry.Name())

		if err := l.LoadLangFile(locale, path); err != nil {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}
	}

	return nil
}

// LoadLangFile loads a single language file
func (l *Localizer) LoadLangFile(locale, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var translations map[string]string
	if err := json.Unmarshal(data, &translations); err != nil {
		return fmt.Errorf("invalid JSON in %s: %w", path, err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.langs[locale] = translations

	return nil
}

// Translate returns the translation for a key in a given locale.
// Falls back to the fallback locale if not found.
// Supports {0}, {1}, etc. positional placeholders.
func (l *Localizer) Translate(locale, key string, args ...interface{}) string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Try requested locale
	if translations, ok := l.langs[locale]; ok {
		if tmpl, ok := translations[key]; ok {
			return formatPlaceholders(tmpl, args...)
		}
	}

	// Try fallback locale
	if locale != l.fallback {
		if translations, ok := l.langs[l.fallback]; ok {
			if tmpl, ok := translations[key]; ok {
				return formatPlaceholders(tmpl, args...)
			}
		}
	}

	// Return key as-is if no translation found
	return key
}

// ForPlayer returns a translated string for a player's locale.
// Currently defaults to "en" until ConVar cl_language integration.
func (l *Localizer) ForPlayer(player *Player, key string, args ...interface{}) string {
	locale := "en" // Default; future: detect from player ConVar
	return l.Translate(locale, key, args...)
}

// formatPlaceholders replaces {0}, {1}, etc. with formatted args
func formatPlaceholders(template string, args ...interface{}) string {
	result := template
	for i, arg := range args {
		placeholder := fmt.Sprintf("{%d}", i)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprint(arg))
	}
	return result
}

package templates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/corrreia/gostrike/internal/manager"
	"github.com/corrreia/gostrike/internal/shared"
)

var (
	mu        sync.RWMutex
	templates = make(map[string]*Template)
)

// templateDirs lists possible locations for template JSON files.
var templateDirs = []string{
	"csgo/addons/gostrike/configs/templates",
	"/home/steam/cs2-dedicated/game/csgo/addons/gostrike/configs/templates",
	"addons/gostrike/configs/templates",
	"configs/templates",
}

// LoadTemplates reads all JSON files from the templates directory.
func LoadTemplates() {
	mu.Lock()
	defer mu.Unlock()

	templates = make(map[string]*Template)

	var dir string
	for _, d := range templateDirs {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			dir = d
			break
		}
	}

	if dir == "" {
		shared.LogDebug("Templates", "No templates directory found")
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		shared.LogWarning("Templates", "Failed to read templates dir: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			shared.LogWarning("Templates", "Failed to read %s: %v", path, err)
			continue
		}

		var tmpl Template
		if err := json.Unmarshal(data, &tmpl); err != nil {
			shared.LogWarning("Templates", "Failed to parse %s: %v", path, err)
			continue
		}

		if tmpl.Name == "" {
			tmpl.Name = strings.TrimSuffix(entry.Name(), ".json")
		}

		templates[tmpl.Name] = &tmpl
		shared.LogDebug("Templates", "Loaded template: %s", tmpl.Name)
	}

	shared.LogInfo("Templates", "Loaded %d template(s)", len(templates))
}

// GetTemplate returns a raw template by name.
func GetTemplate(name string) *Template {
	mu.RLock()
	defer mu.RUnlock()
	return templates[name]
}

// ListTemplates returns all loaded template names.
func ListTemplates() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(templates))
	for name := range templates {
		names = append(names, name)
	}
	return names
}

// GetAllTemplates returns all loaded templates.
func GetAllTemplates() map[string]*Template {
	mu.RLock()
	defer mu.RUnlock()

	result := make(map[string]*Template, len(templates))
	for k, v := range templates {
		result[k] = v
	}
	return result
}

// ResolveTemplate walks the extends chain and merges plugins/convars.
func ResolveTemplate(name string) (*ResolvedTemplate, error) {
	mu.RLock()
	defer mu.RUnlock()

	return resolveTemplateLocked(name)
}

func resolveTemplateLocked(name string) (*ResolvedTemplate, error) {
	// Walk the chain, collecting templates from base to leaf
	var chain []*Template
	visited := make(map[string]bool)

	current := name
	for current != "" {
		if visited[current] {
			return nil, fmt.Errorf("circular inheritance detected at template '%s'", current)
		}
		visited[current] = true

		tmpl, ok := templates[current]
		if !ok {
			return nil, fmt.Errorf("template '%s' not found", current)
		}

		chain = append(chain, tmpl)
		current = tmpl.Extends
	}

	// Merge from base (last) to leaf (first)
	resolved := &ResolvedTemplate{
		Name:    name,
		ConVars: make(map[string]string),
	}

	for i := len(chain) - 1; i >= 0; i-- {
		tmpl := chain[i]
		resolved.Chain = append(resolved.Chain, tmpl.Name)

		// Merge plugins (dedup)
		for _, p := range tmpl.Plugins {
			if !contains(resolved.Plugins, p) {
				resolved.Plugins = append(resolved.Plugins, p)
			}
		}

		// Merge convars (child overrides parent)
		for k, v := range tmpl.ConVars {
			resolved.ConVars[k] = v
		}

		// Map: child overrides parent
		if tmpl.Map != "" {
			resolved.Map = tmpl.Map
		}

		// Description: use leaf's description
		if tmpl.Description != "" {
			resolved.Description = tmpl.Description
		}
	}

	return resolved, nil
}

// ApplyTemplate resolves a template and applies it:
// 1. Compute diff of plugins to load/unload
// 2. Unload extras, load needed
// 3. Set convars
// 4. Change map if specified
func ApplyTemplate(name string, executeCommand func(cmd string), setConVar func(name, value string)) error {
	resolved, err := ResolveTemplate(name)
	if err != nil {
		return err
	}

	// Get currently loaded plugins
	loaded := manager.GetPlugins()
	loadedSlugs := make(map[string]bool)
	for _, p := range loaded {
		if p.State == manager.PluginStateLoaded {
			loadedSlugs[p.Slug] = true
		}
	}

	// Compute desired set
	desiredSlugs := make(map[string]bool, len(resolved.Plugins))
	for _, slug := range resolved.Plugins {
		desiredSlugs[slug] = true
	}

	// Unload plugins not in the desired set
	for slug := range loadedSlugs {
		if !desiredSlugs[slug] {
			if err := manager.UnloadPlugin(slug); err != nil {
				shared.LogWarning("Templates", "Failed to unload %s: %v", slug, err)
			}
		}
	}

	// Load plugins that should be loaded
	for _, slug := range resolved.Plugins {
		if !loadedSlugs[slug] {
			if err := manager.LoadPlugin(slug); err != nil {
				shared.LogWarning("Templates", "Failed to load %s: %v", slug, err)
			}
		}
	}

	// Set convars
	if setConVar != nil {
		for k, v := range resolved.ConVars {
			setConVar(k, v)
		}
	}

	// Change map
	if resolved.Map != "" && executeCommand != nil {
		executeCommand("changelevel " + resolved.Map)
	}

	shared.LogInfo("Templates", "Applied template: %s", name)
	return nil
}

func contains(s []string, item string) bool {
	for _, v := range s {
		if v == item {
			return true
		}
	}
	return false
}

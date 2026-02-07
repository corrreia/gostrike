package templates

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	httpmod "github.com/corrreia/gostrike/internal/modules/http"
	"github.com/corrreia/gostrike/internal/runtime"
	"github.com/corrreia/gostrike/internal/shared"
)

const templatePermission = "gostrike.admin.templates"

// ApplyFunc is the function signature for applying template effects
// (convar setting, map change). Set during initialization.
var (
	ExecuteCommandFunc func(cmd string)
	SetConVarFunc      func(name, value string)
)

// RegisterTemplateAPI registers template HTTP endpoints and chat commands.
func RegisterTemplateAPI() {
	LoadTemplates()

	mod := httpmod.Get()
	if mod != nil && mod.IsRunning() {
		mod.RegisterHandler("GET", "/api/templates", handleListTemplates)
		mod.RegisterHandler("GET", "/api/templates/*", handleGetTemplate)
		mod.RegisterHandler("POST", "/api/templates/*", handleApplyTemplate)
	}

	err := runtime.RegisterChatCommand(runtime.ChatCommand{
		Name:        "template",
		Description: "Apply server templates",
		Usage:       "<list|apply|info> [name]",
		MinArgs:     1,
		Permission:  templatePermission,
		Handler:     handleTemplateCommand,
	})
	if err != nil {
		shared.LogWarning("Templates", "Failed to register !template command: %v", err)
	}
}

// HTTP Handlers

func handleListTemplates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	all := GetAllTemplates()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":     len(all),
		"templates": all,
	})
}

func handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	name := strings.TrimPrefix(r.URL.Path, "/api/templates/")
	if name == "" {
		handleListTemplates(w, r)
		return
	}

	// Strip trailing /apply for POST handling
	name = strings.TrimSuffix(name, "/apply")

	resolved, err := ResolveTemplate(name)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(resolved)
}

func handleApplyTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimPrefix(r.URL.Path, "/api/templates/")
	name := strings.TrimSuffix(path, "/apply")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "template name required"})
		return
	}

	if err := ApplyTemplate(name, ExecuteCommandFunc, SetConVarFunc); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":       true,
		"template": name,
	})
}

// Chat Command Handler

func handleTemplateCommand(slot int, args []string) bool {
	action := strings.ToLower(args[0])

	switch action {
	case "list":
		names := ListTemplates()
		replyToPlayer(slot, "Templates (%d):", len(names))
		for _, name := range names {
			tmpl := GetTemplate(name)
			desc := ""
			if tmpl != nil && tmpl.Description != "" {
				desc = " - " + tmpl.Description
			}
			replyToPlayer(slot, "  %s%s", name, desc)
		}
		return true

	case "apply":
		if len(args) < 2 {
			replyToPlayer(slot, "Usage: !template apply <name>")
			return true
		}
		if err := ApplyTemplate(args[1], ExecuteCommandFunc, SetConVarFunc); err != nil {
			replyToPlayer(slot, "[ERROR] %v", err)
		} else {
			replyToPlayer(slot, "Template '%s' applied successfully", args[1])
		}
		return true

	case "info":
		if len(args) < 2 {
			replyToPlayer(slot, "Usage: !template info <name>")
			return true
		}
		resolved, err := ResolveTemplate(args[1])
		if err != nil {
			replyToPlayer(slot, "[ERROR] %v", err)
			return true
		}
		replyToPlayer(slot, "Template: %s", resolved.Name)
		if resolved.Description != "" {
			replyToPlayer(slot, "  %s", resolved.Description)
		}
		replyToPlayer(slot, "  Plugins: %s", strings.Join(resolved.Plugins, ", "))
		if len(resolved.ConVars) > 0 {
			replyToPlayer(slot, "  ConVars: %d", len(resolved.ConVars))
		}
		if resolved.Map != "" {
			replyToPlayer(slot, "  Map: %s", resolved.Map)
		}
		if len(resolved.Chain) > 1 {
			replyToPlayer(slot, "  Chain: %s", strings.Join(resolved.Chain, " -> "))
		}
		return true

	default:
		// Treat as shorthand: !template <name> == !template apply <name>
		if err := ApplyTemplate(action, ExecuteCommandFunc, SetConVarFunc); err != nil {
			replyToPlayer(slot, "[ERROR] %v", err)
		} else {
			replyToPlayer(slot, "Template '%s' applied successfully", action)
		}
		return true
	}
}

// replyToPlayerFunc uses the manager's reply function (avoids import cycle with bridge).
var replyToPlayerFunc func(slot int, message string)

// SetReplyFunc sets the reply function for template chat commands.
func SetReplyFunc(fn func(slot int, message string)) {
	replyToPlayerFunc = fn
}

func replyToPlayer(slot int, format string, args ...interface{}) {
	if replyToPlayerFunc != nil {
		replyToPlayerFunc(slot, fmt.Sprintf(format, args...))
	}
}

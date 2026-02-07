package manager

import (
	"encoding/json"
	"net/http"
	"strings"

	httpmod "github.com/corrreia/gostrike/internal/modules/http"
	"github.com/corrreia/gostrike/internal/shared"
)

// RegisterPluginAPI registers plugin management HTTP endpoints.
// Called after modules are initialized.
func RegisterPluginAPI() {
	mod := httpmod.Get()
	if mod == nil || !mod.IsRunning() {
		shared.LogDebug("PluginManager", "HTTP module not available, skipping API registration")
		return
	}

	mod.RegisterHandler("POST", "/api/admin/plugins/*", handlePluginAction)
	mod.RegisterHandler("GET", "/api/admin/plugins", handlePluginList)
}

func handlePluginList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	list := GetPluginList()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":   len(list),
		"plugins": list,
	})
}

func handlePluginAction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse /{slug}/{action} from path after /api/admin/plugins/
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/plugins/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "expected /api/admin/plugins/{slug}/{action}"})
		return
	}

	slug := parts[0]
	action := parts[1]

	var err error
	switch action {
	case "load":
		err = LoadPlugin(slug)
	case "unload":
		err = UnloadPlugin(slug)
	case "reload":
		err = ReloadPluginBySlug(slug)
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "unknown action: " + action})
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	info := GetPluginBySlug(slug)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":     true,
		"action": action,
		"plugin": info,
	})
}

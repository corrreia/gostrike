package permissions

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	httpmod "github.com/corrreia/gostrike/internal/modules/http"
)

// maxJSONBodyBytes is the maximum allowed size for JSON request bodies (1 MB).
const maxJSONBodyBytes = 1 << 20

// registerAPI registers all HTTP endpoints for the permissions module.
// Called during Init(). Routes are safe to register before the HTTP server starts.
func registerAPI() {
	mod := httpmod.Get()
	if mod == nil {
		return
	}

	// ── Roles ──────────────────────────────────────────────────

	// GET /api/permissions/roles — list all roles
	mod.RegisterHandler("GET", "/api/permissions/roles", apiListRoles)

	// POST /api/permissions/roles — create a role
	mod.RegisterHandler("POST", "/api/permissions/roles", apiCreateRole)

	// GET /api/permissions/roles/* — get role by name
	// PUT /api/permissions/roles/* — update role
	// DELETE /api/permissions/roles/* — delete role
	mod.RegisterHandler("GET", "/api/permissions/roles/*", apiGetRole)
	mod.RegisterHandler("PUT", "/api/permissions/roles/*", apiUpdateRole)
	mod.RegisterHandler("DELETE", "/api/permissions/roles/*", apiDeleteRole)

	// POST /api/permissions/roles/*/permissions — add permission to role
	// We use a single wildcard handler and parse the sub-path
	mod.RegisterHandler("POST", "/api/permissions/role-permissions/*", apiAddRolePermission)
	mod.RegisterHandler("DELETE", "/api/permissions/role-permissions/*", apiRemoveRolePermission)

	// ── Players ────────────────────────────────────────────────

	// GET /api/permissions/players — list all players
	mod.RegisterHandler("GET", "/api/permissions/players", apiListPlayers)

	// POST /api/permissions/players — create/update player
	mod.RegisterHandler("POST", "/api/permissions/players", apiUpsertPlayer)

	// GET /api/permissions/players/* — get player by steamID
	// DELETE /api/permissions/players/* — remove player
	mod.RegisterHandler("GET", "/api/permissions/players/*", apiGetPlayer)
	mod.RegisterHandler("DELETE", "/api/permissions/players/*", apiDeletePlayer)

	// Player roles
	mod.RegisterHandler("POST", "/api/permissions/player-roles/*", apiAddPlayerRole)
	mod.RegisterHandler("DELETE", "/api/permissions/player-roles/*", apiRemovePlayerRole)

	// Player permissions
	mod.RegisterHandler("POST", "/api/permissions/player-permissions/*", apiAddPlayerPermission)
	mod.RegisterHandler("DELETE", "/api/permissions/player-permissions/*", apiRemovePlayerPermission)

	// ── Utility ────────────────────────────────────────────────

	// GET /api/permissions/registered — all plugin-registered permissions
	mod.RegisterHandler("GET", "/api/permissions/registered", apiRegistered)

	// POST /api/permissions/check — check a permission
	mod.RegisterHandler("POST", "/api/permissions/check", apiCheck)

	// POST /api/permissions/reload — reload cache from DB
	mod.RegisterHandler("POST", "/api/permissions/reload", apiReload)
}

// ── helpers ──────────────────────────────────────────────────

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonErr(w http.ResponseWriter, status int, msg string) {
	jsonResponse(w, status, map[string]string{"error": msg})
}

// flexUint64 unmarshals from both JSON number and JSON string, avoiding
// precision loss for values that exceed JavaScript's Number.MAX_SAFE_INTEGER.
type flexUint64 uint64

func (f *flexUint64) UnmarshalJSON(b []byte) error {
	// String: "76561198012345678"
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid uint64 string: %s", s)
		}
		*f = flexUint64(v)
		return nil
	}
	// Number: 76561198012345678
	var v uint64
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*f = flexUint64(v)
	return nil
}

func readJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	lr := io.LimitReader(r.Body, maxJSONBodyBytes+1)
	if err := json.NewDecoder(lr).Decode(v); err != nil {
		return err
	}
	// If there's still data beyond the limit, the body was too large.
	// We read one extra byte: if Decode consumed exactly maxJSONBodyBytes+1
	// or more, the limit was exceeded. Check by trying to read one more byte.
	if _, err := lr.Read(make([]byte, 1)); err == nil {
		return fmt.Errorf("request body too large (max %d bytes)", maxJSONBodyBytes)
	}
	return nil
}

// extractPathParam extracts the value after the given prefix from the URL path.
// e.g., extractPathParam("/api/permissions/roles/admin", "/api/permissions/roles/") returns "admin"
func extractPathParam(path, prefix string) string {
	if strings.HasPrefix(path, prefix) {
		return strings.TrimPrefix(path, prefix)
	}
	return ""
}

func parseSteamIDParam(path, prefix string) (uint64, error) {
	s := extractPathParam(path, prefix)
	if s == "" {
		return 0, fmt.Errorf("missing steam_id")
	}
	return strconv.ParseUint(s, 10, 64)
}

// ── Role handlers ────────────────────────────────────────────

func apiListRoles(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	roles, err := pm.GetAllRoles()
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	type roleResp struct {
		Name        string   `json:"name"`
		DisplayName string   `json:"display_name"`
		Immunity    int      `json:"immunity"`
		Permissions []string `json:"permissions"`
	}
	out := make([]roleResp, len(roles))
	for i, r := range roles {
		perms := r.Permissions
		if perms == nil {
			perms = []string{}
		}
		out[i] = roleResp{r.Name, r.DisplayName, r.Immunity, perms}
	}
	jsonResponse(w, http.StatusOK,map[string]interface{}{"count": len(out), "roles": out})
}

func apiCreateRole(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	var req struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Immunity    int    `json:"immunity"`
	}
	if err := readJSON(r, &req); err != nil {
		jsonErr(w, 400, "invalid JSON")
		return
	}
	if req.Name == "" {
		jsonErr(w, 400, "name is required")
		return
	}
	role, err := pm.CreateRole(req.Name, req.DisplayName, req.Immunity)
	if err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"name": role.Name, "display_name": role.DisplayName, "immunity": role.Immunity,
	})
}

func apiGetRole(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	name := extractPathParam(r.URL.Path, "/api/permissions/roles/")
	if name == "" {
		jsonErr(w, 400, "role name required")
		return
	}
	role, err := pm.GetRoleByName(name)
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	if role == nil {
		jsonErr(w, 404, "role not found")
		return
	}
	perms := role.Permissions
	if perms == nil {
		perms = []string{}
	}
	jsonResponse(w, http.StatusOK,map[string]interface{}{
		"name": role.Name, "display_name": role.DisplayName,
		"immunity": role.Immunity, "permissions": perms,
	})
}

func apiUpdateRole(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	name := extractPathParam(r.URL.Path, "/api/permissions/roles/")
	if name == "" {
		jsonErr(w, 400, "role name required")
		return
	}
	var req struct {
		DisplayName string `json:"display_name"`
		Immunity    int    `json:"immunity"`
	}
	if err := readJSON(r, &req); err != nil {
		jsonErr(w, 400, "invalid JSON")
		return
	}
	if err := pm.UpdateRole(name, req.DisplayName, req.Immunity); err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK,map[string]string{"status": "updated"})
}

func apiDeleteRole(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	name := extractPathParam(r.URL.Path, "/api/permissions/roles/")
	if name == "" {
		jsonErr(w, 400, "role name required")
		return
	}
	if err := pm.DeleteRole(name); err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK,map[string]string{"status": "deleted"})
}

func apiAddRolePermission(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	roleName := extractPathParam(r.URL.Path, "/api/permissions/role-permissions/")
	if roleName == "" {
		jsonErr(w, 400, "role name required")
		return
	}
	var req struct {
		Permission string `json:"permission"`
	}
	if err := readJSON(r, &req); err != nil {
		jsonErr(w, 400, "invalid JSON")
		return
	}
	if req.Permission == "" {
		jsonErr(w, 400, "permission is required")
		return
	}
	if err := pm.AddRolePermission(roleName, req.Permission); err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, map[string]string{"status": "added"})
}

func apiRemoveRolePermission(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	// Path: /api/permissions/role-permissions/<role>
	roleName := extractPathParam(r.URL.Path, "/api/permissions/role-permissions/")
	if roleName == "" {
		jsonErr(w, 400, "role name required")
		return
	}
	var req struct {
		Permission string `json:"permission"`
	}
	if err := readJSON(r, &req); err != nil {
		jsonErr(w, 400, "invalid JSON")
		return
	}
	if req.Permission == "" {
		jsonErr(w, 400, "permission is required")
		return
	}
	if err := pm.RemoveRolePermission(roleName, req.Permission); err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK,map[string]string{"status": "removed"})
}

// ── Player handlers ──────────────────────────────────────────

func apiListPlayers(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	players, err := pm.GetAllPlayers()
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	type playerResp struct {
		SteamID     uint64   `json:"steam_id"`
		Name        string   `json:"name"`
		Immunity    int      `json:"immunity"`
		ExpiresAt   int64    `json:"expires_at"`
		Roles       []string `json:"roles"`
		Permissions []string `json:"permissions"`
	}
	out := make([]playerResp, len(players))
	for i, p := range players {
		roles := p.Roles
		if roles == nil {
			roles = []string{}
		}
		perms := p.Permissions
		if perms == nil {
			perms = []string{}
		}
		out[i] = playerResp{p.SteamID, p.Name, p.Immunity, p.ExpiresAt, roles, perms}
	}
	jsonResponse(w, http.StatusOK,map[string]interface{}{"count": len(out), "players": out})
}

func apiUpsertPlayer(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	var req struct {
		SteamID   flexUint64 `json:"steam_id"`
		Name      string     `json:"name"`
		Immunity  int        `json:"immunity"`
		ExpiresAt int64      `json:"expires_at"`
	}
	if err := readJSON(r, &req); err != nil {
		jsonErr(w, 400, "invalid JSON")
		return
	}
	if req.SteamID == 0 {
		jsonErr(w, 400, "steam_id is required")
		return
	}
	if err := pm.UpsertPlayer(uint64(req.SteamID), req.Name, req.Immunity, req.ExpiresAt); err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func apiGetPlayer(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	steamID, err := parseSteamIDParam(r.URL.Path, "/api/permissions/players/")
	if err != nil {
		jsonErr(w, 400, "invalid steam_id")
		return
	}
	player, err := pm.GetPlayer(steamID)
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	if player == nil {
		jsonErr(w, 404, "player not found")
		return
	}
	roles := player.Roles
	if roles == nil {
		roles = []string{}
	}
	perms := player.Permissions
	if perms == nil {
		perms = []string{}
	}
	effective := pm.GetEffectivePermissions(steamID)
	if effective == nil {
		effective = []string{}
	}
	jsonResponse(w, http.StatusOK,map[string]interface{}{
		"steam_id":              player.SteamID,
		"name":                  player.Name,
		"immunity":              player.Immunity,
		"expires_at":            player.ExpiresAt,
		"roles":                 roles,
		"permissions":           perms,
		"effective_permissions": effective,
	})
}

func apiDeletePlayer(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	steamID, err := parseSteamIDParam(r.URL.Path, "/api/permissions/players/")
	if err != nil {
		jsonErr(w, 400, "invalid steam_id")
		return
	}
	if err := pm.DeletePlayer(steamID); err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK,map[string]string{"status": "deleted"})
}

func apiAddPlayerRole(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	steamID, err := parseSteamIDParam(r.URL.Path, "/api/permissions/player-roles/")
	if err != nil {
		jsonErr(w, 400, "invalid steam_id")
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := readJSON(r, &req); err != nil {
		jsonErr(w, 400, "invalid JSON")
		return
	}
	if req.Role == "" {
		jsonErr(w, 400, "role is required")
		return
	}
	if err := pm.AddPlayerRole(steamID, req.Role); err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, map[string]string{"status": "added"})
}

func apiRemovePlayerRole(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	steamID, err := parseSteamIDParam(r.URL.Path, "/api/permissions/player-roles/")
	if err != nil {
		jsonErr(w, 400, "invalid steam_id")
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := readJSON(r, &req); err != nil {
		jsonErr(w, 400, "invalid JSON")
		return
	}
	if req.Role == "" {
		jsonErr(w, 400, "role is required")
		return
	}
	if err := pm.RemovePlayerRole(steamID, req.Role); err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK,map[string]string{"status": "removed"})
}

func apiAddPlayerPermission(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	steamID, err := parseSteamIDParam(r.URL.Path, "/api/permissions/player-permissions/")
	if err != nil {
		jsonErr(w, 400, "invalid steam_id")
		return
	}
	var req struct {
		Permission string `json:"permission"`
	}
	if err := readJSON(r, &req); err != nil {
		jsonErr(w, 400, "invalid JSON")
		return
	}
	if req.Permission == "" {
		jsonErr(w, 400, "permission is required")
		return
	}
	if err := pm.AddPlayerPermission(steamID, req.Permission); err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, map[string]string{"status": "added"})
}

func apiRemovePlayerPermission(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	steamID, err := parseSteamIDParam(r.URL.Path, "/api/permissions/player-permissions/")
	if err != nil {
		jsonErr(w, 400, "invalid steam_id")
		return
	}
	var req struct {
		Permission string `json:"permission"`
	}
	if err := readJSON(r, &req); err != nil {
		jsonErr(w, 400, "invalid JSON")
		return
	}
	if req.Permission == "" {
		jsonErr(w, 400, "permission is required")
		return
	}
	if err := pm.RemovePlayerPermission(steamID, req.Permission); err != nil {
		jsonErr(w, 400, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK,map[string]string{"status": "removed"})
}

// ── Utility handlers ─────────────────────────────────────────

func apiRegistered(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	perms := pm.GetRegisteredPermissions()
	type permEntry struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	out := make([]permEntry, 0, len(perms))
	for name, desc := range perms {
		out = append(out, permEntry{name, desc})
	}
	jsonResponse(w, http.StatusOK,map[string]interface{}{"count": len(out), "permissions": out})
}

func apiCheck(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	var req struct {
		SteamID    flexUint64 `json:"steam_id"`
		Permission string     `json:"permission"`
	}
	if err := readJSON(r, &req); err != nil {
		jsonErr(w, 400, "invalid JSON")
		return
	}
	if req.SteamID == 0 || req.Permission == "" {
		jsonErr(w, 400, "steam_id and permission are required")
		return
	}
	has := pm.HasPermission(uint64(req.SteamID), req.Permission)
	jsonResponse(w, http.StatusOK,map[string]interface{}{
		"steam_id":       req.SteamID,
		"permission":     req.Permission,
		"has_permission": has,
	})
}

func apiReload(w http.ResponseWriter, r *http.Request) {
	pm := Get()
	if pm == nil {
		jsonErr(w, 500, "permissions not initialized")
		return
	}
	if err := pm.Reload(); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK,map[string]string{"status": "reloaded"})
}

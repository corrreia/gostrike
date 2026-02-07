package permissions

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/corrreia/gostrike/internal/modules"
	"github.com/corrreia/gostrike/internal/shared"
)

func init() {
	modules.Register(New())
}

// Module implements the string-based permissions module.
type Module struct {
	mu     sync.RWMutex
	db     *sql.DB
	cache  *permCache
	loaded bool
}

var instance *Module

// New creates a new permissions module.
func New() *Module {
	if instance != nil {
		return instance
	}
	instance = &Module{
		cache: newPermCache(),
	}
	return instance
}

// Get returns the singleton instance.
func Get() *Module {
	return instance
}

func (m *Module) Name() string    { return "Permissions" }
func (m *Module) Version() string { return "2.0.0" }
func (m *Module) Priority() int   { return 10 }

// Init opens the SQLite database, runs migrations, seeds defaults, and loads cache.
func (m *Module) Init() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.loaded {
		return nil
	}

	db, err := openDB()
	if err != nil {
		return fmt.Errorf("permissions: %w", err)
	}
	m.db = db

	if err := migrate(db); err != nil {
		db.Close()
		return fmt.Errorf("permissions migrate: %w", err)
	}

	if err := seed(db); err != nil {
		db.Close()
		return fmt.Errorf("permissions seed: %w", err)
	}

	if err := m.reloadCacheLocked(); err != nil {
		db.Close()
		return fmt.Errorf("permissions cache: %w", err)
	}

	m.loaded = true

	// Register HTTP API endpoints
	registerAPI()

	admins, roles := m.statsLocked()
	shared.LogInfo("Permissions", "Initialized (players=%d, roles=%d)", admins, roles)
	return nil
}

// Shutdown closes the database.
func (m *Module) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db != nil {
		m.db.Close()
		m.db = nil
	}
	m.cache.clear()
	m.loaded = false
	return nil
}

// Reload reloads the cache from the database.
func (m *Module) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reloadCacheLocked()
}

func (m *Module) reloadCacheLocked() error {
	roles, err := storeGetAllRoles(m.db)
	if err != nil {
		return err
	}
	players, err := storeGetAllPlayers(m.db)
	if err != nil {
		return err
	}
	m.cache.loadFromDB(roles, players)
	return nil
}

func (m *Module) statsLocked() (players int, roles int) {
	m.cache.mu.RLock()
	defer m.cache.mu.RUnlock()
	return len(m.cache.players), len(m.cache.roles)
}

// ============================================================
// Permission Checks (fast path — cache only)
// ============================================================

// HasPermission checks if a steamID has a specific permission string.
func (m *Module) HasPermission(steamID uint64, permission string) bool {
	return m.cache.hasPermission(steamID, permission)
}

// IsAdmin checks if a steamID has any permissions.
func (m *Module) IsAdmin(steamID uint64) bool {
	return m.cache.isAdmin(steamID)
}

// GetImmunity returns the effective immunity for a steamID.
func (m *Module) GetImmunity(steamID uint64) int {
	return m.cache.getImmunity(steamID)
}

// CanTarget checks if source can target destination based on immunity.
func (m *Module) CanTarget(sourceSteamID, targetSteamID uint64) bool {
	// If source has "*" (root), always allowed
	if m.cache.hasPermission(sourceSteamID, "*") {
		return true
	}
	return m.cache.getImmunity(sourceSteamID) >= m.cache.getImmunity(targetSteamID)
}

// GetEffectivePermissions returns all resolved permissions for a player.
func (m *Module) GetEffectivePermissions(steamID uint64) []string {
	return m.cache.getEffectivePermissions(steamID)
}

// RegisterPermission records a plugin-declared permission.
func (m *Module) RegisterPermission(name, description string) {
	m.cache.registerPermission(name, description)
}

// GetRegisteredPermissions returns all plugin-registered permissions.
func (m *Module) GetRegisteredPermissions() map[string]string {
	return m.cache.getRegistered()
}

// ============================================================
// Role CRUD (write-through: DB + cache reload)
// ============================================================

func (m *Module) GetAllRoles() ([]dbRole, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.db == nil {
		return nil, fmt.Errorf("permissions not initialized")
	}
	return storeGetAllRoles(m.db)
}

func (m *Module) GetRoleByName(name string) (*dbRole, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.db == nil {
		return nil, fmt.Errorf("permissions not initialized")
	}
	return storeGetRoleByName(m.db, name)
}

func (m *Module) CreateRole(name, displayName string, immunity int) (*dbRole, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.db == nil {
		return nil, fmt.Errorf("permissions not initialized")
	}
	role, err := storeCreateRole(m.db, name, displayName, immunity)
	if err != nil {
		return nil, err
	}
	if err := m.reloadCacheLocked(); err != nil {
		return nil, err
	}
	return role, nil
}

func (m *Module) UpdateRole(name, displayName string, immunity int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.db == nil {
		return fmt.Errorf("permissions not initialized")
	}
	if err := storeUpdateRole(m.db, name, displayName, immunity); err != nil {
		return err
	}
	return m.reloadCacheLocked()
}

func (m *Module) DeleteRole(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.db == nil {
		return fmt.Errorf("permissions not initialized")
	}
	if err := storeDeleteRole(m.db, name); err != nil {
		return err
	}
	return m.reloadCacheLocked()
}

func (m *Module) AddRolePermission(roleName, perm string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.db == nil {
		return fmt.Errorf("permissions not initialized")
	}
	role, err := storeGetRoleByName(m.db, roleName)
	if err != nil {
		return err
	}
	if role == nil {
		return fmt.Errorf("role not found: %s", roleName)
	}
	if err := storeAddRolePermission(m.db, role.ID, perm); err != nil {
		return err
	}
	return m.reloadCacheLocked()
}

func (m *Module) RemoveRolePermission(roleName, perm string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.db == nil {
		return fmt.Errorf("permissions not initialized")
	}
	role, err := storeGetRoleByName(m.db, roleName)
	if err != nil {
		return err
	}
	if role == nil {
		return fmt.Errorf("role not found: %s", roleName)
	}
	if err := storeRemoveRolePermission(m.db, role.ID, perm); err != nil {
		return err
	}
	return m.reloadCacheLocked()
}

// ============================================================
// Player CRUD (write-through: DB + cache reload)
// ============================================================

func (m *Module) GetAllPlayers() ([]dbPlayer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.db == nil {
		return nil, fmt.Errorf("permissions not initialized")
	}
	return storeGetAllPlayers(m.db)
}

func (m *Module) GetPlayer(steamID uint64) (*dbPlayer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.db == nil {
		return nil, fmt.Errorf("permissions not initialized")
	}
	return storeGetPlayer(m.db, steamID)
}

func (m *Module) UpsertPlayer(steamID uint64, name string, immunity int, expiresAt int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.db == nil {
		return fmt.Errorf("permissions not initialized")
	}
	if err := storeUpsertPlayer(m.db, steamID, name, immunity, expiresAt); err != nil {
		return err
	}
	return m.reloadCacheLocked()
}

func (m *Module) DeletePlayer(steamID uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.db == nil {
		return fmt.Errorf("permissions not initialized")
	}
	if err := storeDeletePlayer(m.db, steamID); err != nil {
		return err
	}
	return m.reloadCacheLocked()
}

func (m *Module) AddPlayerRole(steamID uint64, roleName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.db == nil {
		return fmt.Errorf("permissions not initialized")
	}
	if err := storeAddPlayerRole(m.db, steamID, roleName); err != nil {
		return err
	}
	return m.reloadCacheLocked()
}

func (m *Module) RemovePlayerRole(steamID uint64, roleName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.db == nil {
		return fmt.Errorf("permissions not initialized")
	}
	if err := storeRemovePlayerRole(m.db, steamID, roleName); err != nil {
		return err
	}
	return m.reloadCacheLocked()
}

func (m *Module) AddPlayerPermission(steamID uint64, perm string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.db == nil {
		return fmt.Errorf("permissions not initialized")
	}
	if err := storeAddPlayerPermission(m.db, steamID, perm); err != nil {
		return err
	}
	return m.reloadCacheLocked()
}

func (m *Module) RemovePlayerPermission(steamID uint64, perm string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.db == nil {
		return fmt.Errorf("permissions not initialized")
	}
	if err := storeRemovePlayerPermission(m.db, steamID, perm); err != nil {
		return err
	}
	return m.reloadCacheLocked()
}

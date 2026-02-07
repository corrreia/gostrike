package permissions

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// openDB opens (or creates) the permissions SQLite database.
func openDB() (*sql.DB, error) {
	// Try several paths to match the CS2 runtime working directory
	dataDirs := []string{
		"csgo/addons/gostrike/data",
		"/home/steam/cs2-dedicated/game/csgo/addons/gostrike/data",
		"addons/gostrike/data",
		"data",
	}

	var dbPath string
	for _, dir := range dataDirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			dbPath = filepath.Join(dir, "permissions.db")
			break
		}
	}
	if dbPath == "" {
		// Fallback: create data/ in cwd
		if err := os.MkdirAll("data", 0755); err != nil {
			return nil, fmt.Errorf("create data dir: %w", err)
		}
		dbPath = "data/permissions.db"
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}

// migrate creates all required tables.
func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS roles (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			name         TEXT    NOT NULL UNIQUE,
			display_name TEXT    NOT NULL DEFAULT '',
			immunity     INTEGER NOT NULL DEFAULT 0,
			created_at   INTEGER NOT NULL DEFAULT 0,
			updated_at   INTEGER NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS role_permissions (
			role_id    INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
			permission TEXT    NOT NULL,
			UNIQUE(role_id, permission)
		);

		CREATE TABLE IF NOT EXISTS players (
			steam_id   INTEGER PRIMARY KEY,
			name       TEXT    NOT NULL DEFAULT '',
			immunity   INTEGER NOT NULL DEFAULT 0,
			expires_at INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS player_roles (
			steam_id INTEGER NOT NULL REFERENCES players(steam_id) ON DELETE CASCADE,
			role_id  INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
			UNIQUE(steam_id, role_id)
		);

		CREATE TABLE IF NOT EXISTS player_permissions (
			steam_id   INTEGER NOT NULL REFERENCES players(steam_id) ON DELETE CASCADE,
			permission TEXT    NOT NULL,
			UNIQUE(steam_id, permission)
		);
	`)
	return err
}

// seed inserts default roles if the roles table is empty.
func seed(db *sql.DB) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM roles").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil // already seeded
	}

	now := time.Now().Unix()

	type seedRole struct {
		name        string
		displayName string
		immunity    int
		perms       []string
	}

	defaults := []seedRole{
		{"root", "Root", 100, []string{"*"}},
		{"admin", "Administrator", 80, []string{"gostrike.*"}},
		{"moderator", "Moderator", 50, []string{
			"gostrike.kick", "gostrike.ban", "gostrike.slay", "gostrike.chat",
		}},
		{"vip", "VIP", 10, []string{"gostrike.reservation"}},
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, r := range defaults {
		res, err := tx.Exec(
			"INSERT INTO roles (name, display_name, immunity, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			r.name, r.displayName, r.immunity, now, now,
		)
		if err != nil {
			return fmt.Errorf("seed role %s: %w", r.name, err)
		}
		roleID, _ := res.LastInsertId()

		for _, perm := range r.perms {
			if _, err := tx.Exec(
				"INSERT INTO role_permissions (role_id, permission) VALUES (?, ?)",
				roleID, perm,
			); err != nil {
				return fmt.Errorf("seed perm %s/%s: %w", r.name, perm, err)
			}
		}
	}

	return tx.Commit()
}

// ============================================================
// Role CRUD
// ============================================================

type dbRole struct {
	ID          int64
	Name        string
	DisplayName string
	Immunity    int
	Permissions []string
	CreatedAt   int64
	UpdatedAt   int64
}

func storeGetAllRoles(db *sql.DB) ([]dbRole, error) {
	rows, err := db.Query("SELECT id, name, display_name, immunity, created_at, updated_at FROM roles ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []dbRole
	for rows.Next() {
		var r dbRole
		if err := rows.Scan(&r.ID, &r.Name, &r.DisplayName, &r.Immunity, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load permissions for each role
	for i := range roles {
		perms, err := storeGetRolePermissions(db, roles[i].ID)
		if err != nil {
			return nil, err
		}
		roles[i].Permissions = perms
	}

	return roles, nil
}

func storeGetRoleByName(db *sql.DB, name string) (*dbRole, error) {
	var r dbRole
	err := db.QueryRow(
		"SELECT id, name, display_name, immunity, created_at, updated_at FROM roles WHERE name = ?",
		name,
	).Scan(&r.ID, &r.Name, &r.DisplayName, &r.Immunity, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	perms, err := storeGetRolePermissions(db, r.ID)
	if err != nil {
		return nil, err
	}
	r.Permissions = perms
	return &r, nil
}

func storeCreateRole(db *sql.DB, name, displayName string, immunity int) (*dbRole, error) {
	now := time.Now().Unix()
	res, err := db.Exec(
		"INSERT INTO roles (name, display_name, immunity, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		name, displayName, immunity, now, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &dbRole{
		ID: id, Name: name, DisplayName: displayName,
		Immunity: immunity, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func storeUpdateRole(db *sql.DB, name, displayName string, immunity int) error {
	now := time.Now().Unix()
	res, err := db.Exec(
		"UPDATE roles SET display_name = ?, immunity = ?, updated_at = ? WHERE name = ?",
		displayName, immunity, now, name,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("role not found: %s", name)
	}
	return nil
}

func storeDeleteRole(db *sql.DB, name string) error {
	res, err := db.Exec("DELETE FROM roles WHERE name = ?", name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("role not found: %s", name)
	}
	return nil
}

func storeGetRolePermissions(db *sql.DB, roleID int64) ([]string, error) {
	rows, err := db.Query("SELECT permission FROM role_permissions WHERE role_id = ?", roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return perms, nil
}

func storeAddRolePermission(db *sql.DB, roleID int64, perm string) error {
	_, err := db.Exec(
		"INSERT OR IGNORE INTO role_permissions (role_id, permission) VALUES (?, ?)",
		roleID, perm,
	)
	return err
}

func storeRemoveRolePermission(db *sql.DB, roleID int64, perm string) error {
	_, err := db.Exec(
		"DELETE FROM role_permissions WHERE role_id = ? AND permission = ?",
		roleID, perm,
	)
	return err
}

// ============================================================
// Player CRUD
// ============================================================

type dbPlayer struct {
	SteamID     uint64
	Name        string
	Immunity    int
	ExpiresAt   int64
	Roles       []string // role names
	Permissions []string // direct permissions
	CreatedAt   int64
	UpdatedAt   int64
}

func storeGetAllPlayers(db *sql.DB) ([]dbPlayer, error) {
	rows, err := db.Query("SELECT steam_id, name, immunity, expires_at, created_at, updated_at FROM players ORDER BY steam_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []dbPlayer
	for rows.Next() {
		var p dbPlayer
		if err := rows.Scan(&p.SteamID, &p.Name, &p.Immunity, &p.ExpiresAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range players {
		roles, err := storeGetPlayerRoles(db, players[i].SteamID)
		if err != nil {
			return nil, err
		}
		players[i].Roles = roles

		perms, err := storeGetPlayerPermissions(db, players[i].SteamID)
		if err != nil {
			return nil, err
		}
		players[i].Permissions = perms
	}

	return players, nil
}

func storeGetPlayer(db *sql.DB, steamID uint64) (*dbPlayer, error) {
	var p dbPlayer
	err := db.QueryRow(
		"SELECT steam_id, name, immunity, expires_at, created_at, updated_at FROM players WHERE steam_id = ?",
		steamID,
	).Scan(&p.SteamID, &p.Name, &p.Immunity, &p.ExpiresAt, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	roles, err := storeGetPlayerRoles(db, steamID)
	if err != nil {
		return nil, err
	}
	p.Roles = roles

	perms, err := storeGetPlayerPermissions(db, steamID)
	if err != nil {
		return nil, err
	}
	p.Permissions = perms

	return &p, nil
}

func storeUpsertPlayer(db *sql.DB, steamID uint64, name string, immunity int, expiresAt int64) error {
	now := time.Now().Unix()
	_, err := db.Exec(`
		INSERT INTO players (steam_id, name, immunity, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(steam_id) DO UPDATE SET
			name = excluded.name,
			immunity = excluded.immunity,
			expires_at = excluded.expires_at,
			updated_at = ?
	`, steamID, name, immunity, expiresAt, now, now, now)
	return err
}

func storeDeletePlayer(db *sql.DB, steamID uint64) error {
	_, err := db.Exec("DELETE FROM players WHERE steam_id = ?", steamID)
	return err
}

func storeGetPlayerRoles(db *sql.DB, steamID uint64) ([]string, error) {
	rows, err := db.Query(`
		SELECT r.name FROM player_roles pr
		JOIN roles r ON r.id = pr.role_id
		WHERE pr.steam_id = ?
	`, steamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		roles = append(roles, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return roles, nil
}

func storeAddPlayerRole(db *sql.DB, steamID uint64, roleName string) error {
	res, err := db.Exec(`
		INSERT OR IGNORE INTO player_roles (steam_id, role_id)
		SELECT ?, id FROM roles WHERE name = ?
	`, steamID, roleName)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("role not found: %s", roleName)
	}
	return nil
}

func storeRemovePlayerRole(db *sql.DB, steamID uint64, roleName string) error {
	_, err := db.Exec(`
		DELETE FROM player_roles WHERE steam_id = ? AND role_id = (SELECT id FROM roles WHERE name = ?)
	`, steamID, roleName)
	return err
}

func storeGetPlayerPermissions(db *sql.DB, steamID uint64) ([]string, error) {
	rows, err := db.Query("SELECT permission FROM player_permissions WHERE steam_id = ?", steamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return perms, nil
}

func storeAddPlayerPermission(db *sql.DB, steamID uint64, perm string) error {
	_, err := db.Exec(
		"INSERT OR IGNORE INTO player_permissions (steam_id, permission) VALUES (?, ?)",
		steamID, perm,
	)
	return err
}

func storeRemovePlayerPermission(db *sql.DB, steamID uint64, perm string) error {
	_, err := db.Exec(
		"DELETE FROM player_permissions WHERE steam_id = ? AND permission = ?",
		steamID, perm,
	)
	return err
}

package permissions

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// testDB creates an in-memory SQLite database for testing.
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

// ── matchPermission tests ─────────────────────────────────────

func TestMatchPermission(t *testing.T) {
	tests := []struct {
		have string
		want string
		ok   bool
	}{
		{"*", "gostrike.kick", true},
		{"*", "anything", true},
		{"gostrike.*", "gostrike.kick", true},
		{"gostrike.*", "gostrike.ban", true},
		{"gostrike.*", "example.give", false},
		{"gostrike.kick", "gostrike.kick", true},
		{"gostrike.kick", "gostrike.ban", false},
		{"example.*", "example.give", true},
		{"example.*", "gostrike.kick", false},
		{"a.b.*", "a.b.c", true},
		{"a.b.*", "a.b.c.d", true},
		{"a.b.*", "a.x", false},
	}
	for _, tc := range tests {
		got := matchPermission(tc.have, tc.want)
		if got != tc.ok {
			t.Errorf("matchPermission(%q, %q) = %v, want %v", tc.have, tc.want, got, tc.ok)
		}
	}
}

// ── store tests ───────────────────────────────────────────────

func TestStoreSeedAndRoles(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	if err := seed(db); err != nil {
		t.Fatal(err)
	}

	roles, err := storeGetAllRoles(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 4 {
		t.Fatalf("expected 4 seeded roles, got %d", len(roles))
	}

	// Check root role
	root, err := storeGetRoleByName(db, "root")
	if err != nil {
		t.Fatal(err)
	}
	if root == nil {
		t.Fatal("root role not found")
	}
	if len(root.Permissions) != 1 || root.Permissions[0] != "*" {
		t.Errorf("root permissions = %v, want [*]", root.Permissions)
	}
	if root.Immunity != 100 {
		t.Errorf("root immunity = %d, want 100", root.Immunity)
	}
}

func TestStoreSeedIdempotent(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	if err := seed(db); err != nil {
		t.Fatal(err)
	}
	if err := seed(db); err != nil {
		t.Fatal(err)
	}

	roles, _ := storeGetAllRoles(db)
	if len(roles) != 4 {
		t.Fatalf("expected 4 roles after double seed, got %d", len(roles))
	}
}

func TestStoreRoleCRUD(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	// Create
	role, err := storeCreateRole(db, "testrole", "Test Role", 42)
	if err != nil {
		t.Fatal(err)
	}
	if role.Name != "testrole" || role.Immunity != 42 {
		t.Errorf("unexpected role: %+v", role)
	}

	// Add permission
	if err := storeAddRolePermission(db, role.ID, "test.perm"); err != nil {
		t.Fatal(err)
	}

	// Get
	got, err := storeGetRoleByName(db, "testrole")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("role not found")
	}
	if len(got.Permissions) != 1 || got.Permissions[0] != "test.perm" {
		t.Errorf("permissions = %v, want [test.perm]", got.Permissions)
	}

	// Update
	if err := storeUpdateRole(db, "testrole", "Updated", 99); err != nil {
		t.Fatal(err)
	}
	got, _ = storeGetRoleByName(db, "testrole")
	if got.DisplayName != "Updated" || got.Immunity != 99 {
		t.Errorf("update failed: %+v", got)
	}

	// Remove permission
	if err := storeRemoveRolePermission(db, role.ID, "test.perm"); err != nil {
		t.Fatal(err)
	}
	got, _ = storeGetRoleByName(db, "testrole")
	if len(got.Permissions) != 0 {
		t.Errorf("expected 0 permissions after remove, got %d", len(got.Permissions))
	}

	// Delete
	if err := storeDeleteRole(db, "testrole"); err != nil {
		t.Fatal(err)
	}
	got, _ = storeGetRoleByName(db, "testrole")
	if got != nil {
		t.Error("role should be nil after delete")
	}
}

func TestStorePlayerCRUD(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	seed(db)

	steamID := uint64(76561198012345678)

	// Upsert
	if err := storeUpsertPlayer(db, steamID, "TestPlayer", 50, 0); err != nil {
		t.Fatal(err)
	}

	// Get
	p, err := storeGetPlayer(db, steamID)
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("player not found")
	}
	if p.Name != "TestPlayer" || p.Immunity != 50 {
		t.Errorf("unexpected player: %+v", p)
	}

	// Add role
	if err := storeAddPlayerRole(db, steamID, "admin"); err != nil {
		t.Fatal(err)
	}
	p, _ = storeGetPlayer(db, steamID)
	if len(p.Roles) != 1 || p.Roles[0] != "admin" {
		t.Errorf("roles = %v, want [admin]", p.Roles)
	}

	// Add direct permission
	if err := storeAddPlayerPermission(db, steamID, "custom.perm"); err != nil {
		t.Fatal(err)
	}
	p, _ = storeGetPlayer(db, steamID)
	if len(p.Permissions) != 1 || p.Permissions[0] != "custom.perm" {
		t.Errorf("permissions = %v, want [custom.perm]", p.Permissions)
	}

	// Remove role
	if err := storeRemovePlayerRole(db, steamID, "admin"); err != nil {
		t.Fatal(err)
	}
	p, _ = storeGetPlayer(db, steamID)
	if len(p.Roles) != 0 {
		t.Errorf("expected 0 roles after remove, got %d", len(p.Roles))
	}

	// Delete
	if err := storeDeletePlayer(db, steamID); err != nil {
		t.Fatal(err)
	}
	p, _ = storeGetPlayer(db, steamID)
	if p != nil {
		t.Error("player should be nil after delete")
	}
}

// ── cache tests ───────────────────────────────────────────────

func TestCachePermissionCheck(t *testing.T) {
	c := newPermCache()

	roles := []dbRole{
		{ID: 1, Name: "admin", Immunity: 80, Permissions: []string{"gostrike.*"}},
		{ID: 2, Name: "vip", Immunity: 10, Permissions: []string{"gostrike.reservation"}},
	}
	players := []dbPlayer{
		{
			SteamID:     100,
			Name:        "Admin",
			Immunity:    0,
			Roles:       []string{"admin"},
			Permissions: []string{"custom.direct"},
		},
		{
			SteamID:     200,
			Name:        "VIP",
			Immunity:    0,
			Roles:       []string{"vip"},
			Permissions: nil,
		},
		{
			SteamID:     300,
			Name:        "Nobody",
			Immunity:    0,
			Roles:       nil,
			Permissions: nil,
		},
	}
	c.loadFromDB(roles, players)

	// Admin has gostrike.*
	if !c.hasPermission(100, "gostrike.kick") {
		t.Error("admin should have gostrike.kick")
	}
	if !c.hasPermission(100, "gostrike.ban") {
		t.Error("admin should have gostrike.ban")
	}
	if c.hasPermission(100, "example.give") {
		t.Error("admin should NOT have example.give")
	}
	// Admin has direct custom.direct
	if !c.hasPermission(100, "custom.direct") {
		t.Error("admin should have custom.direct")
	}

	// VIP has only reservation
	if !c.hasPermission(200, "gostrike.reservation") {
		t.Error("VIP should have gostrike.reservation")
	}
	if c.hasPermission(200, "gostrike.kick") {
		t.Error("VIP should NOT have gostrike.kick")
	}

	// Nobody has nothing
	if c.hasPermission(300, "gostrike.kick") {
		t.Error("nobody should NOT have gostrike.kick")
	}

	// Unknown player
	if c.hasPermission(999, "anything") {
		t.Error("unknown player should NOT have any permission")
	}
}

func TestCacheImmunity(t *testing.T) {
	c := newPermCache()

	roles := []dbRole{
		{ID: 1, Name: "admin", Immunity: 80, Permissions: nil},
		{ID: 2, Name: "vip", Immunity: 10, Permissions: nil},
	}
	players := []dbPlayer{
		{SteamID: 100, Immunity: 0, Roles: []string{"admin"}},
		{SteamID: 200, Immunity: 0, Roles: []string{"vip"}},
		{SteamID: 300, Immunity: 90, Roles: []string{"vip"}}, // personal immunity > role
		{SteamID: 400, Immunity: 0, Roles: nil},
	}
	c.loadFromDB(roles, players)

	if got := c.getImmunity(100); got != 80 {
		t.Errorf("admin immunity = %d, want 80", got)
	}
	if got := c.getImmunity(200); got != 10 {
		t.Errorf("vip immunity = %d, want 10", got)
	}
	if got := c.getImmunity(300); got != 90 {
		t.Errorf("personal immunity = %d, want 90", got)
	}
	if got := c.getImmunity(400); got != 0 {
		t.Errorf("nobody immunity = %d, want 0", got)
	}
}

func TestCacheIsAdmin(t *testing.T) {
	c := newPermCache()

	roles := []dbRole{{ID: 1, Name: "admin", Permissions: []string{"gostrike.*"}}}
	players := []dbPlayer{
		{SteamID: 100, Roles: []string{"admin"}},
		{SteamID: 200, Roles: nil, Permissions: []string{"custom.perm"}},
		{SteamID: 300, Roles: nil, Permissions: nil},
	}
	c.loadFromDB(roles, players)

	if !c.isAdmin(100) {
		t.Error("player with role should be admin")
	}
	if !c.isAdmin(200) {
		t.Error("player with direct perm should be admin")
	}
	if c.isAdmin(300) {
		t.Error("player with no perms/roles should NOT be admin")
	}
}

func TestCacheRegisteredPermissions(t *testing.T) {
	c := newPermCache()

	c.registerPermission("example.give", "Give weapons")
	c.registerPermission("example.hp", "Set health")

	reg := c.getRegistered()
	if len(reg) != 2 {
		t.Fatalf("expected 2 registered, got %d", len(reg))
	}
	if reg["example.give"] != "Give weapons" {
		t.Errorf("unexpected description: %s", reg["example.give"])
	}
}

// ── SteamID tests ─────────────────────────────────────────────

func TestParseSteamID(t *testing.T) {
	tests := []struct {
		input string
		want  uint64
		err   bool
	}{
		{"76561198012345678", 76561198012345678, false},
		{"STEAM_0:0:26039975", 76561198012345678, false},
		{"[U:1:52079950]", 76561198012345678, false},
		{"invalid", 0, true},
		{"", 0, true},
		{"99999999999999999999", 0, true}, // overflow
	}
	for _, tc := range tests {
		got, err := ParseSteamID(tc.input)
		if tc.err {
			if err == nil {
				t.Errorf("ParseSteamID(%q) expected error", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseSteamID(%q) error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseSteamID(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestFormatSteamID2(t *testing.T) {
	// Valid SteamID64
	s, err := FormatSteamID2(76561198012345678)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != "STEAM_0:0:26039975" {
		t.Errorf("FormatSteamID2 = %q, want STEAM_0:0:26039975", s)
	}

	// Below base offset
	_, err = FormatSteamID2(12345)
	if err == nil {
		t.Error("FormatSteamID2(12345) expected error for id below base offset")
	}
}

func TestFormatSteamID3(t *testing.T) {
	// Valid SteamID64
	s, err := FormatSteamID3(76561198012345678)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != "[U:1:52079950]" {
		t.Errorf("FormatSteamID3 = %q, want [U:1:52079950]", s)
	}

	// Below base offset
	_, err = FormatSteamID3(0)
	if err == nil {
		t.Error("FormatSteamID3(0) expected error for id below base offset")
	}
}

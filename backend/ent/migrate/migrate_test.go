package migrate_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/loomi-labs/arco/backend/ent"
	_ "github.com/loomi-labs/arco/backend/ent/runtime"
	"github.com/loomi-labs/arco/backend/util"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

// migrationValidators maps each migration filename (without .sql) to a
// validation function. When a new migration is added without a corresponding
// validator, TestMigrationCoverage fails — forcing you to write one.
var migrationValidators = map[string]func(t *testing.T, ctx context.Context, db *sql.DB, client *ent.Client){
	"20241202145510_init":                                         validateInit,
	"20241202193640_default_settings":                             validateDefaultSettings,
	"20250221150025_add_collapse_state":                           validateCollapseState,
	"20250606081555_gen":                                          validateAuthTables,
	"20250608160803_gen":                                          validateAuthSessionsRecreate,
	"20250609111245_gen":                                          validateUsersRecreate,
	"20250728130000_repository_schema_update":                     validateRepositorySchemaUpdate,
	"20251002142957_gen":                                          validateRepositoryNormalize,
	"20251120094352_gen":                                          validateSettingsExpertMode,
	"20251120124959_gen":                                          validateProfileCompressionMode,
	"20251121155656_gen":                                          validateRepositoryCheckFields,
	"20251124144244_gen":                                          validateArchiveFKConsolidation,
	"20251128144447_gen":                                          validateArchiveComment,
	"20251204090416_gen":                                          validateExcludeCaches,
	"20251204185732_gen":                                          validateSettingsTransitions,
	"20251207203648_gen":                                          validateSettingsDropWelcome,
	"20251215134854_gen":                                          validateArchiveWarningMessage,
	"20251217094826_gen":                                          validateHasPassword,
	"20251217094827_gen":                                          validateDropPasswordAndTokens,
	"20251219131416_gen":                                          validateUniqueProfileNames,
	"20251231113739_gen":                                          validateMacfuseWarning,
	"20260127174049_add_full_disk_access_warning_dismissed":       validateFullDiskAccessWarning,
	"20260202123657_gen":                                          validateIntervalMinutes,
	"20260327153722_gen":                                          validateFeedbackPrompt,
	"20260331130829_gen":                                          validateAnalytics,
}

// TestMigrationCoverage ensures every migration file has a registered validator.
func TestMigrationCoverage(t *testing.T) {
	entries, err := os.ReadDir("migrations")
	if err != nil {
		t.Fatalf("failed to read migrations directory: %v", err)
	}

	var sqlFiles []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sql" {
			continue
		}
		sqlFiles = append(sqlFiles, strings.TrimSuffix(e.Name(), ".sql"))
	}
	sort.Strings(sqlFiles)

	// Check for migrations without validators.
	var missing []string
	for _, f := range sqlFiles {
		if _, ok := migrationValidators[f]; !ok {
			missing = append(missing, f)
		}
	}

	// Check for validators without migrations.
	var extra []string
	fileSet := toSet(sqlFiles)
	for name := range migrationValidators {
		if !fileSet[name] {
			extra = append(extra, name)
		}
	}

	if len(missing) > 0 {
		t.Errorf("migrations without validators: %v\n"+
			"Add a validator to migrationValidators in migrate_test.go.", missing)
	}
	if len(extra) > 0 {
		t.Errorf("validators for non-existent migrations: %v\n"+
			"Remove stale entries from migrationValidators.", extra)
	}

	// Verify seed files in testdata/ correspond to real migration versions.
	versionSet := make(map[string]bool, len(sqlFiles))
	for _, f := range sqlFiles {
		version, _, _ := strings.Cut(f, "_")
		versionSet[version] = true
	}

	seedEntries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("failed to read testdata directory: %v", err)
	}
	for _, e := range seedEntries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sql" {
			continue
		}
		version := strings.TrimSuffix(e.Name(), ".sql")
		if !versionSet[version] {
			t.Errorf("seed file %s has no matching migration — remove or rename it", e.Name())
		}
	}
}

// TestSchemaCompleteness ensures all schema entities are covered by the migration test.
func TestSchemaCompleteness(t *testing.T) {
	knownEntities := []string{
		"analyticsevent.go",
		"archive.go",
		"authsession.go",
		"backupprofile.go",
		"backupschedule.go",
		"cloudrepository.go",
		"notification.go",
		"pruningrule.go",
		"repository.go",
		"settings.go",
		"user.go",
	}

	schemaDir := filepath.Join("..", "schema")
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		t.Fatalf("failed to read schema directory: %v", err)
	}

	var found []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" {
			continue
		}
		found = append(found, e.Name())
	}

	sort.Strings(knownEntities)
	sort.Strings(found)

	var added, removed []string
	known := toSet(knownEntities)
	foundSet := toSet(found)

	for _, f := range found {
		if !known[f] {
			added = append(added, f)
		}
	}
	for _, k := range knownEntities {
		if !foundSet[k] {
			removed = append(removed, k)
		}
	}

	if len(added) > 0 {
		t.Errorf("new schema entities detected: %v\n"+
			"Update knownEntities, seed data, and the migration validators.", added)
	}
	if len(removed) > 0 {
		t.Errorf("schema entities removed: %v\n"+
			"Update knownEntities in TestSchemaCompleteness.", removed)
	}
}

// relationship represents a foreign key in the database schema, including its actions.
type relationship struct {
	Table      string
	Column     string
	References string
	OnUpdate   string
	OnDelete   string
}

func (r relationship) String() string {
	return fmt.Sprintf("%s.%s → %s (update=%s, delete=%s)", r.Table, r.Column, r.References, r.OnUpdate, r.OnDelete)
}

// knownRelationships lists all FK relationships that must exist after all migrations.
// TestRelationshipCoverage fails if a FK exists in the DB that isn't listed here,
// forcing you to add seed data and validation for new relationships.
var knownRelationships = []relationship{
	{Table: "archives", Column: "archive_repository", References: "repositories", OnUpdate: "NO ACTION", OnDelete: "CASCADE"},
	{Table: "archives", Column: "backup_profile_archives", References: "backup_profiles", OnUpdate: "NO ACTION", OnDelete: "SET NULL"},
	{Table: "backup_profile_repositories", Column: "backup_profile_id", References: "backup_profiles", OnUpdate: "NO ACTION", OnDelete: "CASCADE"},
	{Table: "backup_profile_repositories", Column: "repository_id", References: "repositories", OnUpdate: "NO ACTION", OnDelete: "CASCADE"},
	{Table: "backup_schedules", Column: "backup_profile_backup_schedule", References: "backup_profiles", OnUpdate: "NO ACTION", OnDelete: "CASCADE"},
	{Table: "notifications", Column: "notification_backup_profile", References: "backup_profiles", OnUpdate: "NO ACTION", OnDelete: "CASCADE"},
	{Table: "notifications", Column: "notification_repository", References: "repositories", OnUpdate: "NO ACTION", OnDelete: "CASCADE"},
	{Table: "pruning_rules", Column: "backup_profile_pruning_rule", References: "backup_profiles", OnUpdate: "NO ACTION", OnDelete: "CASCADE"},
	{Table: "repositories", Column: "cloud_repository_repository", References: "cloud_repositories", OnUpdate: "NO ACTION", OnDelete: "SET NULL"},
}

// TestRelationshipCoverage discovers all FK definitions in the migrated schema
// and fails if any FK is not in knownRelationships (or vice versa).
func TestRelationshipCoverage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	dbSource := "file:" + dbPath + "?_fk=1"

	db, err := sql.Open("sqlite3", dbSource)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Apply all migrations.
	migrationsFS := os.DirFS("migrations")
	gooseMigrations := &util.CustomFS{
		FS:     migrationsFS,
		Prefix: "-- +goose Up\n-- +goose StatementBegin\n",
		Suffix: "\n-- +goose StatementEnd\n",
	}
	goose.SetBaseFS(gooseMigrations)
	if err := goose.SetDialect(dialect.SQLite); err != nil {
		t.Fatalf("failed to set dialect: %v", err)
	}
	if err := goose.Up(db, "."); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	// Discover all tables (excluding internal ones).
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT IN ('goose_db_version', 'sqlite_sequence')`)
	if err != nil {
		t.Fatalf("failed to query tables: %v", err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("failed to scan table name: %v", err)
		}
		tables = append(tables, name)
	}
	rows.Close()

	// Discover all foreign keys via PRAGMA.
	var discovered []relationship
	for _, table := range tables {
		fkRows, err := db.Query(fmt.Sprintf("PRAGMA foreign_key_list(%s)", table))
		if err != nil {
			t.Fatalf("failed to query FKs for %s: %v", table, err)
		}
		for fkRows.Next() {
			var id, seq int
			var refTable, from, to, onUpdate, onDelete, match string
			if err := fkRows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
				t.Fatalf("failed to scan FK for %s: %v", table, err)
			}
			discovered = append(discovered, relationship{Table: table, Column: from, References: refTable, OnUpdate: onUpdate, OnDelete: onDelete})
		}
		fkRows.Close()
	}

	// Build lookup sets.
	knownSet := make(map[string]bool, len(knownRelationships))
	for _, r := range knownRelationships {
		knownSet[r.String()] = true
	}
	discoveredSet := make(map[string]bool, len(discovered))
	for _, r := range discovered {
		discoveredSet[r.String()] = true
	}

	// Check for new FKs not in the known set.
	var newFKs []string
	for _, r := range discovered {
		if !knownSet[r.String()] {
			newFKs = append(newFKs, r.String())
		}
	}

	// Check for stale entries in the known set.
	var staleFKs []string
	for _, r := range knownRelationships {
		if !discoveredSet[r.String()] {
			staleFKs = append(staleFKs, r.String())
		}
	}

	if len(newFKs) > 0 {
		t.Errorf("new foreign keys not in knownRelationships: %v\n"+
			"Add them to knownRelationships, add seed data, and validate the relationship in a migration validator.", newFKs)
	}
	if len(staleFKs) > 0 {
		t.Errorf("stale entries in knownRelationships (FK no longer exists): %v\n"+
			"Remove them from knownRelationships.", staleFKs)
	}
}

// TestMigrationDataIntegrity applies migrations one-by-one, seeding test data
// after each step, then runs every registered validator.
func TestMigrationDataIntegrity(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	dbSource := "file:" + dbPath + "?_fk=1"

	db, err := sql.Open("sqlite3", dbSource)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Setup goose with CustomFS wrapper.
	migrationsFS := os.DirFS("migrations")
	gooseMigrations := &util.CustomFS{
		FS:     migrationsFS,
		Prefix: "-- +goose Up\n-- +goose StatementBegin\n",
		Suffix: "\n-- +goose StatementEnd\n",
	}
	goose.SetBaseFS(gooseMigrations)
	if err := goose.SetDialect(dialect.SQLite); err != nil {
		t.Fatalf("failed to set dialect: %v", err)
	}

	// Apply migrations one-by-one, seeding after each step.
	for _, name := range migrationNames(t) {
		version, _, _ := strings.Cut(name, "_")
		versionInt, err := strconv.ParseInt(version, 10, 64)
		if err != nil {
			t.Fatalf("failed to parse version %q: %v", version, err)
		}

		if err := goose.UpTo(db, ".", versionInt); err != nil {
			t.Fatalf("failed to apply migration %s: %v", name, err)
		}

		seedFile := filepath.Join("testdata", version+".sql")
		seedSQL, err := os.ReadFile(seedFile)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			t.Fatalf("failed to read seed %s: %v", seedFile, err)
		}
		if _, err := db.Exec(string(seedSQL)); err != nil {
			t.Fatalf("failed to execute seed %s: %v", seedFile, err)
		}
		t.Logf("seeded %s", version)
	}

	// Open Ent client.
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))
	defer client.Close()

	ctx := context.Background()

	// Run all validators.
	for name, validate := range migrationValidators {
		t.Run(name, func(t *testing.T) {
			validate(t, ctx, db, client)
		})
	}
}

// --- validators ---

// validateInit checks that all seeded entities exist with correct data.
func validateInit(t *testing.T, ctx context.Context, _ *sql.DB, client *ent.Client) {
	t.Helper()

	// Verify repository count.
	repoCount, err := client.Repository.Query().Count(ctx)
	if err != nil {
		t.Fatalf("failed to count repositories: %v", err)
	}
	if repoCount != 2 {
		t.Errorf("expected 2 repositories, got %d", repoCount)
	}

	// Verify backup profile count.
	profileCount, err := client.BackupProfile.Query().Count(ctx)
	if err != nil {
		t.Fatalf("failed to count backup profiles: %v", err)
	}
	if profileCount != 3 {
		t.Errorf("expected 3 backup profiles, got %d", profileCount)
	}

	// Verify backup_profile_repositories relationships.
	profiles, err := client.BackupProfile.Query().
		WithRepositories().
		Order(ent.Asc("id")).
		All(ctx)
	if err != nil {
		t.Fatalf("failed to query profiles: %v", err)
	}

	expectedReposByProfile := map[int]int{1: 2, 2: 1, 3: 1}
	for _, p := range profiles {
		repos, _ := p.Edges.RepositoriesOrErr()
		if len(repos) != expectedReposByProfile[p.ID] {
			t.Errorf("profile %d: expected %d repos, got %d", p.ID, expectedReposByProfile[p.ID], len(repos))
		}
	}

	// Verify archives.
	archiveCount, err := client.Archive.Query().Count(ctx)
	if err != nil {
		t.Fatalf("failed to count archives: %v", err)
	}
	if archiveCount != 2 {
		t.Errorf("expected 2 archives, got %d", archiveCount)
	}

	// Verify archive relationships (repository + backup_profile edges).
	archives, err := client.Archive.Query().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		t.Fatalf("failed to query archives: %v", err)
	}
	expectedArchiveProfiles := map[int]int{1: 1, 2: 2}
	for _, a := range archives {
		repoID, err := a.QueryRepository().OnlyID(ctx)
		if err != nil {
			t.Fatalf("archive %d: failed to query repository: %v", a.ID, err)
		}
		if repoID != 1 {
			t.Errorf("archive %d: expected repository 1, got %d", a.ID, repoID)
		}

		profileID, err := a.QueryBackupProfile().OnlyID(ctx)
		if err != nil {
			t.Fatalf("archive %d: failed to query backup profile: %v", a.ID, err)
		}
		if profileID != expectedArchiveProfiles[a.ID] {
			t.Errorf("archive %d: expected profile %d, got %d", a.ID, expectedArchiveProfiles[a.ID], profileID)
		}
	}

	// Verify notification relationships (backup_profile + repository edges).
	notifications, err := client.Notification.Query().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		t.Fatalf("failed to query notifications: %v", err)
	}
	if len(notifications) != 2 {
		t.Errorf("expected 2 notifications, got %d", len(notifications))
	}
	expectedNotifProfiles := map[int]int{1: 1, 2: 2}
	for _, n := range notifications {
		profileID, err := n.QueryBackupProfile().OnlyID(ctx)
		if err != nil {
			t.Fatalf("notification %d: failed to query profile: %v", n.ID, err)
		}
		if profileID != expectedNotifProfiles[n.ID] {
			t.Errorf("notification %d: expected profile %d, got %d", n.ID, expectedNotifProfiles[n.ID], profileID)
		}

		repoID, err := n.QueryRepository().OnlyID(ctx)
		if err != nil {
			t.Fatalf("notification %d: failed to query repository: %v", n.ID, err)
		}
		if repoID != 1 {
			t.Errorf("notification %d: expected repository 1, got %d", n.ID, repoID)
		}
	}

	// Verify backup schedule relationships.
	schedules, err := client.BackupSchedule.Query().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		t.Fatalf("failed to query backup schedules: %v", err)
	}
	if len(schedules) != 3 {
		t.Errorf("expected 3 backup schedules, got %d", len(schedules))
	}
	for i, s := range schedules {
		profileID, err := s.QueryBackupProfile().OnlyID(ctx)
		if err != nil {
			t.Fatalf("schedule %d: failed to query profile: %v", s.ID, err)
		}
		if profileID != i+1 {
			t.Errorf("schedule %d: expected profile %d, got %d", s.ID, i+1, profileID)
		}
	}

	// Verify pruning rule relationships.
	rules, err := client.PruningRule.Query().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		t.Fatalf("failed to query pruning rules: %v", err)
	}
	if len(rules) != 3 {
		t.Errorf("expected 3 pruning rules, got %d", len(rules))
	}
	for i, rule := range rules {
		profileID, err := rule.QueryBackupProfile().OnlyID(ctx)
		if err != nil {
			t.Fatalf("rule %d: failed to query profile: %v", rule.ID, err)
		}
		if profileID != i+1 {
			t.Errorf("rule %d: expected profile %d, got %d", rule.ID, i+1, profileID)
		}
	}
}

// validateDefaultSettings checks that the default settings row was inserted.
func validateDefaultSettings(t *testing.T, ctx context.Context, _ *sql.DB, client *ent.Client) {
	t.Helper()

	settings, err := client.Settings.Query().Only(ctx)
	if err != nil {
		t.Fatalf("failed to query settings: %v", err)
	}
	if settings.ID == 0 {
		t.Error("settings row should exist")
	}
}

// validateCollapseState checks that collapse columns were added with correct defaults.
func validateCollapseState(t *testing.T, ctx context.Context, db *sql.DB, _ *ent.Client) {
	t.Helper()

	var dataCollapsed, scheduleCollapsed bool
	err := db.QueryRowContext(ctx,
		`SELECT data_section_collapsed, schedule_section_collapsed FROM backup_profiles WHERE id = 1`).
		Scan(&dataCollapsed, &scheduleCollapsed)
	if err != nil {
		t.Fatalf("failed to query collapse state: %v", err)
	}
	if dataCollapsed {
		t.Error("data_section_collapsed should default to false")
	}
	if scheduleCollapsed {
		t.Error("schedule_section_collapsed should default to false")
	}
}

// validateAuthTables checks that auth_sessions and users tables exist.
func validateAuthTables(t *testing.T, ctx context.Context, _ *sql.DB, client *ent.Client) {
	t.Helper()

	// Users table should be queryable (may be empty).
	_, err := client.User.Query().Count(ctx)
	if err != nil {
		t.Fatalf("users table should exist: %v", err)
	}

	// Auth sessions table should be queryable.
	_, err = client.AuthSession.Query().Count(ctx)
	if err != nil {
		t.Fatalf("auth_sessions table should exist: %v", err)
	}
}

// validateAuthSessionsRecreate checks that auth_sessions was recreated (user_email dropped, tokens moved to users).
func validateAuthSessionsRecreate(t *testing.T, ctx context.Context, db *sql.DB, _ *ent.Client) {
	t.Helper()

	if columnExists(t, db, "auth_sessions", "user_email") {
		t.Error("user_email column should have been removed from auth_sessions")
	}
}

// validateUsersRecreate checks that users table was converted to integer PK and auth_sessions got session_id.
func validateUsersRecreate(t *testing.T, ctx context.Context, db *sql.DB, _ *ent.Client) {
	t.Helper()

	// Verify users.id is INTEGER PRIMARY KEY.
	if !columnIsPK(t, db, "users", "id", "integer") {
		t.Error("users.id should be INTEGER PRIMARY KEY")
	}

	// Verify auth_sessions got session_id column.
	if !columnExists(t, db, "auth_sessions", "session_id") {
		t.Error("session_id column should exist on auth_sessions")
	}
}

// validateRepositorySchemaUpdate checks that location was renamed to url and cloud_repositories table created.
func validateRepositorySchemaUpdate(t *testing.T, ctx context.Context, db *sql.DB, client *ent.Client) {
	t.Helper()

	// Verify 'location' column no longer exists.
	if columnExists(t, db, "repositories", "location") {
		t.Error("'location' column should not exist after migration")
	}

	// Verify url was populated from location.
	repos, err := client.Repository.Query().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		t.Fatalf("failed to query repositories: %v", err)
	}

	expectedURLs := map[int]string{
		1: "ssh://user@host1.example.com:22/~/backup",
		2: "ssh://user@host2.example.com:22/~/backup",
	}
	for _, repo := range repos {
		if repo.URL != expectedURLs[repo.ID] {
			t.Errorf("repo %d: expected url %q, got %q", repo.ID, expectedURLs[repo.ID], repo.URL)
		}
	}

	// Verify cloud_repositories table exists.
	_, err = client.CloudRepository.Query().Count(ctx)
	if err != nil {
		t.Fatalf("cloud_repositories table should exist: %v", err)
	}
}

// validateRepositoryNormalize checks that repo data survived table recreation with normalized structure.
func validateRepositoryNormalize(t *testing.T, ctx context.Context, _ *sql.DB, client *ent.Client) {
	t.Helper()

	repos, err := client.Repository.Query().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		t.Fatalf("failed to query repositories: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repositories, got %d", len(repos))
	}

	// Verify stats survived.
	if repos[0].StatsTotalChunks != 100 {
		t.Errorf("repo 1: expected stats_total_chunks 100, got %d", repos[0].StatsTotalChunks)
	}
	if repos[1].StatsTotalSize != 2048000 {
		t.Errorf("repo 2: expected stats_total_size 2048000, got %d", repos[1].StatsTotalSize)
	}

	// Verify relationships survived recreation with exact repo IDs.
	profiles, err := client.BackupProfile.Query().WithRepositories().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		t.Fatalf("failed to query profiles: %v", err)
	}
	assertProfileRepoIDs(t, profiles)
}

// validateSettingsExpertMode checks that expert_mode and theme columns exist with defaults.
func validateSettingsExpertMode(t *testing.T, ctx context.Context, _ *sql.DB, client *ent.Client) {
	t.Helper()

	settings, err := client.Settings.Query().Only(ctx)
	if err != nil {
		t.Fatalf("failed to query settings: %v", err)
	}
	if settings.ExpertMode != false {
		t.Error("expert_mode should default to false")
	}
	if string(settings.Theme) != "system" {
		t.Errorf("theme should default to 'system', got %q", settings.Theme)
	}
}

// validateProfileCompressionMode checks that compression fields were added with correct defaults.
func validateProfileCompressionMode(t *testing.T, ctx context.Context, _ *sql.DB, client *ent.Client) {
	t.Helper()

	profiles, err := client.BackupProfile.Query().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		t.Fatalf("failed to query profiles: %v", err)
	}

	for _, p := range profiles {
		if string(p.CompressionMode) != "lz4" {
			t.Errorf("profile %d: compression_mode should default to 'lz4', got %q", p.ID, p.CompressionMode)
		}
		if p.AdvancedSectionCollapsed != true {
			t.Errorf("profile %d: advanced_section_collapsed should default to true", p.ID)
		}
	}
}

// validateRepositoryCheckFields checks that integrity check fields replaced next_integrity_check.
func validateRepositoryCheckFields(t *testing.T, ctx context.Context, db *sql.DB, _ *ent.Client) {
	t.Helper()

	if columnExists(t, db, "repositories", "next_integrity_check") {
		t.Error("next_integrity_check should have been replaced")
	}
	for _, col := range []string{"last_quick_check_at", "quick_check_error", "last_full_check_at", "full_check_error"} {
		if !columnExists(t, db, "repositories", col) {
			t.Errorf("%s column should exist on repositories", col)
		}
	}
}

// validateArchiveFKConsolidation checks that archive FK was consolidated from two columns to one.
func validateArchiveFKConsolidation(t *testing.T, ctx context.Context, db *sql.DB, client *ent.Client) {
	t.Helper()

	// archive_backup_profile column should be gone.
	if columnExists(t, db, "archives", "archive_backup_profile") {
		t.Error("archive_backup_profile column should have been removed")
	}

	// backup_profile_archives should exist and preserve relationships.
	archives, err := client.Archive.Query().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		t.Fatalf("failed to query archives: %v", err)
	}

	// Archive 1 had archive_backup_profile=1, archive 2 had archive_backup_profile=2.
	// COALESCE(archive_backup_profile, backup_profile_archives) should preserve them.
	expectedProfiles := map[int]int{1: 1, 2: 2}
	for _, a := range archives {
		profileID, err := a.QueryBackupProfile().OnlyID(ctx)
		if err != nil {
			t.Fatalf("archive %d: failed to query backup profile: %v", a.ID, err)
		}
		if profileID != expectedProfiles[a.ID] {
			t.Errorf("archive %d: expected profile %d, got %d", a.ID, expectedProfiles[a.ID], profileID)
		}
	}
}

// validateArchiveComment checks that comment column was added to archives.
func validateArchiveComment(t *testing.T, ctx context.Context, db *sql.DB, _ *ent.Client) {
	t.Helper()

	if !columnExists(t, db, "archives", "comment") {
		t.Error("comment column should exist on archives")
	}
}

// validateExcludeCaches checks that exclude_caches column was added to backup_profiles.
func validateExcludeCaches(t *testing.T, ctx context.Context, _ *sql.DB, client *ent.Client) {
	t.Helper()

	profiles, err := client.BackupProfile.Query().All(ctx)
	if err != nil {
		t.Fatalf("failed to query profiles: %v", err)
	}
	for _, p := range profiles {
		if p.ExcludeCaches != false {
			t.Errorf("profile %d: exclude_caches should default to false", p.ID)
		}
	}
}

// validateSettingsTransitions checks that transition/shadow columns were added.
func validateSettingsTransitions(t *testing.T, ctx context.Context, _ *sql.DB, client *ent.Client) {
	t.Helper()

	settings, err := client.Settings.Query().Only(ctx)
	if err != nil {
		t.Fatalf("failed to query settings: %v", err)
	}
	if settings.DisableTransitions != false {
		t.Error("disable_transitions should default to false")
	}
	if settings.DisableShadows != false {
		t.Error("disable_shadows should default to false")
	}
}

// validateSettingsDropWelcome checks that show_welcome was removed from settings.
func validateSettingsDropWelcome(t *testing.T, ctx context.Context, db *sql.DB, client *ent.Client) {
	t.Helper()

	if columnExists(t, db, "settings", "show_welcome") {
		t.Error("show_welcome column should have been removed")
	}

	// Verify settings data survived recreation.
	settings, err := client.Settings.Query().Only(ctx)
	if err != nil {
		t.Fatalf("failed to query settings: %v", err)
	}
	if settings.ID == 0 {
		t.Error("settings row should still exist")
	}
}

// validateArchiveWarningMessage checks that warning_message was added and warning notifications deleted.
func validateArchiveWarningMessage(t *testing.T, ctx context.Context, db *sql.DB, client *ent.Client) {
	t.Helper()

	if !columnExists(t, db, "archives", "warning_message") {
		t.Error("warning_message column should exist on archives")
	}

	// Verify no warning_backup_run notifications remain.
	var count int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE type = 'warning_backup_run'`).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count warning notifications: %v", err)
	}
	if count != 0 {
		t.Errorf("warning_backup_run notifications should have been deleted, found %d", count)
	}
}

// validateHasPassword checks that has_password was set based on existing password data.
func validateHasPassword(t *testing.T, ctx context.Context, _ *sql.DB, client *ent.Client) {
	t.Helper()

	repos, err := client.Repository.Query().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		t.Fatalf("failed to query repositories: %v", err)
	}

	// Both seeded repos had non-empty passwords.
	for _, repo := range repos {
		if !repo.HasPassword {
			t.Errorf("repo %d: has_password should be true (had password in seed data)", repo.ID)
		}
	}
}

// validateDropPasswordAndTokens checks that password column was removed from repositories.
func validateDropPasswordAndTokens(t *testing.T, ctx context.Context, db *sql.DB, client *ent.Client) {
	t.Helper()

	// Password column should be gone from repositories.
	if columnExists(t, db, "repositories", "password") {
		t.Error("password column should have been removed from repositories")
	}

	// Token columns should be gone from users.
	for _, col := range []string{"access_token", "refresh_token"} {
		if columnExists(t, db, "users", col) {
			t.Errorf("%s column should have been removed from users", col)
		}
	}

	// Verify repository data survived the table recreation.
	repos, err := client.Repository.Query().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		t.Fatalf("failed to query repositories: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repositories, got %d", len(repos))
	}

	if repos[0].Name != "Test Repo 1" {
		t.Errorf("repo 1: name mismatch: got %q", repos[0].Name)
	}
	if repos[0].URL != "ssh://user@host1.example.com:22/~/backup" {
		t.Errorf("repo 1: url mismatch: got %q", repos[0].URL)
	}
	if repos[0].StatsTotalChunks != 100 {
		t.Errorf("repo 1: stats_total_chunks mismatch: got %d", repos[0].StatsTotalChunks)
	}
	if repos[1].StatsTotalSize != 2048000 {
		t.Errorf("repo 2: stats_total_size mismatch: got %d", repos[1].StatsTotalSize)
	}

	// Verify relationships survived with exact repo IDs.
	profiles, err := client.BackupProfile.Query().WithRepositories().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		t.Fatalf("failed to query profiles with repos: %v", err)
	}
	assertProfileRepoIDs(t, profiles)
}

// validateUniqueProfileNames checks that duplicate names were renamed and unique index created.
func validateUniqueProfileNames(t *testing.T, ctx context.Context, db *sql.DB, client *ent.Client) {
	t.Helper()

	profiles, err := client.BackupProfile.Query().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		t.Fatalf("failed to query profiles: %v", err)
	}

	// Profile 1 (first "Home Backup") should keep original name.
	if profiles[0].Name != "Home Backup" {
		t.Errorf("profile 1: first occurrence should keep name 'Home Backup', got %q", profiles[0].Name)
	}

	// Profile 2 should be unchanged.
	if profiles[1].Name != "Work Backup" {
		t.Errorf("profile 2: name should be 'Work Backup', got %q", profiles[1].Name)
	}

	// Profile 3 (second "Home Backup") should become "Home Backup (1)".
	if profiles[2].Name != "Home Backup (1)" {
		t.Errorf("profile 3: duplicate should be renamed to 'Home Backup (1)', got %q", profiles[2].Name)
	}

	// All names should be unique.
	seen := make(map[string]int)
	for _, p := range profiles {
		if existingID, ok := seen[p.Name]; ok {
			t.Errorf("profiles %d and %d both have name %q", existingID, p.ID, p.Name)
		}
		seen[p.Name] = p.ID
	}

	// Verify unique index exists on backup_profiles.name.
	if !uniqueIndexExists(t, db, "backup_profiles", "name") {
		t.Error("unique index should exist on backup_profiles.name")
	}
}

// validateMacfuseWarning checks that macfuse_warning_dismissed was added.
func validateMacfuseWarning(t *testing.T, ctx context.Context, _ *sql.DB, client *ent.Client) {
	t.Helper()

	settings, err := client.Settings.Query().Only(ctx)
	if err != nil {
		t.Fatalf("failed to query settings: %v", err)
	}
	if settings.MacfuseWarningDismissed != false {
		t.Error("macfuse_warning_dismissed should default to false")
	}
}

// validateFullDiskAccessWarning checks that full_disk_access_warning_dismissed was added.
func validateFullDiskAccessWarning(t *testing.T, ctx context.Context, _ *sql.DB, client *ent.Client) {
	t.Helper()

	settings, err := client.Settings.Query().Only(ctx)
	if err != nil {
		t.Fatalf("failed to query settings: %v", err)
	}
	if settings.FullDiskAccessWarningDismissed != false {
		t.Error("full_disk_access_warning_dismissed should default to false")
	}
}

// validateIntervalMinutes checks that mode was transformed and interval_minutes added.
func validateIntervalMinutes(t *testing.T, ctx context.Context, _ *sql.DB, client *ent.Client) {
	t.Helper()

	schedules, err := client.BackupSchedule.Query().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		t.Fatalf("failed to query schedules: %v", err)
	}

	if len(schedules) != 3 {
		t.Fatalf("expected 3 schedules, got %d", len(schedules))
	}

	// Schedule 1 was 'hourly' -> should now be 'minute_interval'.
	if string(schedules[0].Mode) != "minute_interval" {
		t.Errorf("schedule 1: mode should be 'minute_interval', got %q", schedules[0].Mode)
	}
	if schedules[0].IntervalMinutes != 60 {
		t.Errorf("schedule 1: interval_minutes should be 60, got %d", schedules[0].IntervalMinutes)
	}

	// Schedule 2 was 'weekly' -> should stay 'weekly'.
	if string(schedules[1].Mode) != "weekly" {
		t.Errorf("schedule 2: mode should be 'weekly', got %q", schedules[1].Mode)
	}

	// Schedule 3 was 'monthly' -> should stay 'monthly'.
	if string(schedules[2].Mode) != "monthly" {
		t.Errorf("schedule 3: mode should be 'monthly', got %q", schedules[2].Mode)
	}

	// All schedules should default to 60 minutes.
	for _, s := range schedules {
		if s.IntervalMinutes != 60 {
			t.Errorf("schedule %d: interval_minutes should default to 60, got %d", s.ID, s.IntervalMinutes)
		}
	}

	// Verify schedule->profile relationships.
	for i, s := range schedules {
		profileID, err := s.QueryBackupProfile().OnlyID(ctx)
		if err != nil {
			t.Fatalf("schedule %d: failed to query profile: %v", s.ID, err)
		}
		if profileID != i+1 {
			t.Errorf("schedule %d: expected profile %d, got %d", s.ID, i+1, profileID)
		}
	}
}

// validateFeedbackPrompt checks that feedback_last_prompted_at was added to settings.
func validateFeedbackPrompt(t *testing.T, ctx context.Context, db *sql.DB, _ *ent.Client) {
	t.Helper()

	if !columnExists(t, db, "settings", "feedback_last_prompted_at") {
		t.Error("feedback_last_prompted_at column should exist on settings")
	}
}

// validateAnalytics checks that usage_logging_enabled, installation_id, and analytics_events table exist.
func validateAnalytics(t *testing.T, ctx context.Context, db *sql.DB, client *ent.Client) {
	t.Helper()

	// Verify usage_logging_enabled is nullable on settings.
	if !columnExists(t, db, "settings", "usage_logging_enabled") {
		t.Error("usage_logging_enabled column should exist on settings")
	}

	// Verify installation_id exists on settings.
	if !columnExists(t, db, "settings", "installation_id") {
		t.Error("installation_id column should exist on settings")
	}

	// Verify analytics_events table exists and is queryable.
	count, err := client.AnalyticsEvent.Query().Count(ctx)
	if err != nil {
		t.Fatalf("analytics_events table should exist: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 analytics events, got %d", count)
	}

	// Verify sent index exists.
	if !indexExists(t, db, "analytics_events", "sent") {
		t.Error("index on sent column should exist on analytics_events")
	}
}

// --- helpers ---

func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}

// migrationNames returns sorted migration filenames (without .sql) from the
// migrations directory.
func migrationNames(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir("migrations")
	if err != nil {
		t.Fatalf("failed to read migrations directory: %v", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sql" {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".sql"))
	}
	sort.Strings(names)
	return names
}

// columnExists checks if a column exists in a SQLite table using PRAGMA table_info.
func columnExists(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()

	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("failed to get table info for %s: %v", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("failed to scan table info: %v", err)
		}
		if name == column {
			return true
		}
	}
	return false
}

// columnIsPK checks if a column is a primary key with the expected type.
func columnIsPK(t *testing.T, db *sql.DB, table, column, expectedType string) bool {
	t.Helper()

	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("failed to get table info for %s: %v", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("failed to scan table info: %v", err)
		}
		if name == column {
			return pk == 1 && strings.EqualFold(colType, expectedType)
		}
	}
	return false
}

// uniqueIndexExists checks if a unique index exists on a specific column of a table.
func uniqueIndexExists(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()

	// Get all indexes on the table.
	idxRows, err := db.Query(fmt.Sprintf("PRAGMA index_list('%s')", table))
	if err != nil {
		t.Fatalf("failed to get index list for %s: %v", table, err)
	}
	defer idxRows.Close()

	for idxRows.Next() {
		var seq int
		var name string
		var unique int
		var origin, partial string
		if err := idxRows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatalf("failed to scan index list: %v", err)
		}
		if unique != 1 {
			continue
		}

		// Check if this unique index covers the target column.
		infoRows, err := db.Query(fmt.Sprintf("PRAGMA index_info('%s')", name))
		if err != nil {
			t.Fatalf("failed to get index info for %s: %v", name, err)
		}
		for infoRows.Next() {
			var seqno, cid int
			var colName string
			if err := infoRows.Scan(&seqno, &cid, &colName); err != nil {
				t.Fatalf("failed to scan index info: %v", err)
			}
			if colName == column {
				infoRows.Close()
				return true
			}
		}
		infoRows.Close()
	}
	return false
}

// indexExists checks if an index exists on a specific column of a table.
func indexExists(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()

	idxRows, err := db.Query(fmt.Sprintf("PRAGMA index_list('%s')", table))
	if err != nil {
		t.Fatalf("failed to get index list for %s: %v", table, err)
	}
	defer idxRows.Close()

	for idxRows.Next() {
		var seq int
		var name string
		var unique int
		var origin, partial string
		if err := idxRows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatalf("failed to scan index list: %v", err)
		}

		infoRows, err := db.Query(fmt.Sprintf("PRAGMA index_info('%s')", name))
		if err != nil {
			t.Fatalf("failed to get index info for %s: %v", name, err)
		}
		for infoRows.Next() {
			var seqno, cid int
			var colName string
			if err := infoRows.Scan(&seqno, &cid, &colName); err != nil {
				t.Fatalf("failed to scan index info: %v", err)
			}
			if colName == column {
				infoRows.Close()
				return true
			}
		}
		infoRows.Close()
	}
	return false
}

// assertProfileRepoIDs verifies that backup profiles have exact expected repository ID sets.
func assertProfileRepoIDs(t *testing.T, profiles []*ent.BackupProfile) {
	t.Helper()

	expectedRepoIDs := map[int][]int{1: {1, 2}, 2: {1}, 3: {2}}
	for _, p := range profiles {
		repos, _ := p.Edges.RepositoriesOrErr()
		gotIDs := make(map[int]bool, len(repos))
		for _, r := range repos {
			gotIDs[r.ID] = true
		}
		expected := expectedRepoIDs[p.ID]
		if len(gotIDs) != len(expected) {
			t.Errorf("profile %d: expected repo IDs %v, got %v", p.ID, expected, gotIDs)
			continue
		}
		for _, id := range expected {
			if !gotIDs[id] {
				t.Errorf("profile %d: expected repo IDs %v, got %v", p.ID, expected, gotIDs)
				break
			}
		}
	}
}

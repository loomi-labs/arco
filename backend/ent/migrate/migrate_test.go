package migrate_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/loomi-labs/arco/backend/ent"
	_ "github.com/loomi-labs/arco/backend/ent/runtime"
	"github.com/loomi-labs/arco/backend/util"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

// TestMigrationDataIntegrity tests that migrations preserve all data and relationships
// It migrates to an intermediate version, inserts test data, then applies remaining migrations
func TestMigrationDataIntegrity(t *testing.T) {
	// Create temporary directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	dbSource := "file:" + dbPath + "?_fk=1"

	// Open database connection
	db, err := sql.Open("sqlite3", dbSource)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Setup goose
	migrationsFS := os.DirFS("migrations")
	gooseMigrations := &util.CustomFS{
		FS:     migrationsFS,
		Prefix: "-- +goose Up\n-- +goose StatementBegin\n",
		Suffix: "\n-- +goose StatementEnd\n",
	}
	goose.SetBaseFS(gooseMigrations)
	if err := goose.SetDialect(dialect.SQLite); err != nil {
		t.Fatalf("Failed to set dialect: %v", err)
	}

	// Step 1: Migrate to intermediate version (20250221150025_add_collapse_state.sql)
	targetVersion := int64(20250221150025)
	if err := goose.UpTo(db, ".", targetVersion); err != nil {
		t.Fatalf("Failed to apply migrations up to %d: %v", targetVersion, err)
	}

	// Step 2: Load seed data
	seedDataPath := filepath.Join("testdata", "seed_data.sql")
	if err := executeSQLFile(db, seedDataPath); err != nil {
		t.Fatalf("Failed to load seed data: %v", err)
	}

	// Step 3: Capture pre-migration state
	preMigrationState, err := captureState(db)
	if err != nil {
		t.Fatalf("Failed to capture pre-migration state: %v", err)
	}

	// Step 4: Apply remaining migrations
	if err := goose.Up(db, "."); err != nil {
		t.Fatalf("Failed to apply remaining migrations: %v", err)
	}

	// Step 5: Validate post-migration state
	if err := validateMigration(db, preMigrationState); err != nil {
		t.Fatalf("Migration validation failed: %v", err)
	}
}

// executeSQLFile executes SQL statements from a file
func executeSQLFile(db *sql.DB, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %w", err)
	}

	_, err = db.Exec(string(content))
	if err != nil {
		return fmt.Errorf("failed to execute SQL: %w", err)
	}

	return nil
}

// MigrationState holds the database state before migration
type MigrationState struct {
	BackupProfileRepositories []ProfileRepositoryRelation
	Archives                  []ArchiveRelation
	Notifications             []NotificationRelation
	RepositoryCount           int
	BackupProfileCount        int
}

type ProfileRepositoryRelation struct {
	BackupProfileID int
	RepositoryID    int
}

type ArchiveRelation struct {
	ID           int
	RepositoryID int
	ProfileID    sql.NullInt64
}

type NotificationRelation struct {
	ID           int
	ProfileID    int
	RepositoryID int
}

// captureState captures the current state of the database
func captureState(db *sql.DB) (*MigrationState, error) {
	state := &MigrationState{}

	// Count repositories
	err := db.QueryRow("SELECT COUNT(*) FROM repositories").Scan(&state.RepositoryCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count repositories: %w", err)
	}

	// Count backup profiles
	err = db.QueryRow("SELECT COUNT(*) FROM backup_profiles").Scan(&state.BackupProfileCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count backup profiles: %w", err)
	}

	// Capture backup_profile_repositories relationships
	rows, err := db.Query("SELECT backup_profile_id, repository_id FROM backup_profile_repositories ORDER BY backup_profile_id, repository_id")
	if err != nil {
		return nil, fmt.Errorf("failed to query backup_profile_repositories: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rel ProfileRepositoryRelation
		if err := rows.Scan(&rel.BackupProfileID, &rel.RepositoryID); err != nil {
			return nil, fmt.Errorf("failed to scan relationship: %w", err)
		}
		state.BackupProfileRepositories = append(state.BackupProfileRepositories, rel)
	}

	// Capture archive relationships
	rows, err = db.Query("SELECT id, archive_repository, archive_backup_profile FROM archives ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("failed to query archives: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var archive ArchiveRelation
		if err := rows.Scan(&archive.ID, &archive.RepositoryID, &archive.ProfileID); err != nil {
			return nil, fmt.Errorf("failed to scan archive: %w", err)
		}
		state.Archives = append(state.Archives, archive)
	}

	// Capture notification relationships
	rows, err = db.Query("SELECT id, notification_backup_profile, notification_repository FROM notifications ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var notif NotificationRelation
		if err := rows.Scan(&notif.ID, &notif.ProfileID, &notif.RepositoryID); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		state.Notifications = append(state.Notifications, notif)
	}

	return state, nil
}

// validateMigration validates that the migration preserved all data and relationships
func validateMigration(db *sql.DB, preMigrationState *MigrationState) error {
	ctx := context.Background()

	// Create Ent client from database connection
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))
	defer client.Close()

	// Check that repository count is preserved
	repoCount, err := client.Repository.Query().Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to count repositories: %w", err)
	}
	if repoCount != preMigrationState.RepositoryCount {
		return fmt.Errorf("repository count mismatch: expected %d, got %d", preMigrationState.RepositoryCount, repoCount)
	}

	// Check that backup profile count is preserved
	profileCount, err := client.BackupProfile.Query().Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to count backup profiles: %w", err)
	}
	if profileCount != preMigrationState.BackupProfileCount {
		return fmt.Errorf("backup profile count mismatch: expected %d, got %d", preMigrationState.BackupProfileCount, profileCount)
	}

	// Validate backup_profile_repositories relationships using Ent edges
	profiles, err := client.BackupProfile.Query().
		WithRepositories().
		Order(ent.Asc("id")).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query backup profiles with repositories: %w", err)
	}

	var postMigrationRels []ProfileRepositoryRelation
	for _, profile := range profiles {
		repos, err := profile.Edges.RepositoriesOrErr()
		if err != nil {
			return fmt.Errorf("failed to get repositories for profile %d: %w", profile.ID, err)
		}
		for _, repo := range repos {
			postMigrationRels = append(postMigrationRels, ProfileRepositoryRelation{
				BackupProfileID: profile.ID,
				RepositoryID:    repo.ID,
			})
		}
	}

	if len(postMigrationRels) != len(preMigrationState.BackupProfileRepositories) {
		return fmt.Errorf("backup_profile_repositories count mismatch: expected %d, got %d",
			len(preMigrationState.BackupProfileRepositories), len(postMigrationRels))
	}

	for i, rel := range postMigrationRels {
		expected := preMigrationState.BackupProfileRepositories[i]
		if rel.BackupProfileID != expected.BackupProfileID || rel.RepositoryID != expected.RepositoryID {
			return fmt.Errorf("backup_profile_repositories mismatch at index %d: expected (%d, %d), got (%d, %d)",
				i, expected.BackupProfileID, expected.RepositoryID, rel.BackupProfileID, rel.RepositoryID)
		}
	}

	// Validate archive relationships
	archives, err := client.Archive.Query().
		Order(ent.Asc("id")).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query archives: %w", err)
	}

	if len(archives) != len(preMigrationState.Archives) {
		return fmt.Errorf("archives count mismatch: expected %d, got %d",
			len(preMigrationState.Archives), len(archives))
	}

	for i, archive := range archives {
		expected := preMigrationState.Archives[i]
		archiveRepoID, err := archive.QueryRepository().OnlyID(ctx)
		if err != nil {
			return fmt.Errorf("failed to get repository ID for archive %d: %w", archive.ID, err)
		}
		if archive.ID != expected.ID || archiveRepoID != expected.RepositoryID {
			return fmt.Errorf("archive relationship mismatch at index %d", i)
		}
	}

	// Validate notification relationships
	notifications, err := client.Notification.Query().
		Order(ent.Asc("id")).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query notifications: %w", err)
	}

	if len(notifications) != len(preMigrationState.Notifications) {
		return fmt.Errorf("notifications count mismatch: expected %d, got %d",
			len(preMigrationState.Notifications), len(notifications))
	}

	for i, notif := range notifications {
		expected := preMigrationState.Notifications[i]
		profileID, err := notif.QueryBackupProfile().OnlyID(ctx)
		if err != nil {
			return fmt.Errorf("failed to get backup profile ID for notification %d: %w", notif.ID, err)
		}
		repoID, err := notif.QueryRepository().OnlyID(ctx)
		if err != nil {
			return fmt.Errorf("failed to get repository ID for notification %d: %w", notif.ID, err)
		}
		if notif.ID != expected.ID || profileID != expected.ProfileID || repoID != expected.RepositoryID {
			return fmt.Errorf("notification relationship mismatch at index %d", i)
		}
	}

	// Validate that 'url' column exists and has data (all repositories should have non-empty URL)
	repos, err := client.Repository.Query().All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query repositories: %w", err)
	}

	urlCount := 0
	for _, repo := range repos {
		if repo.URL != "" {
			urlCount++
		}
	}

	if urlCount != preMigrationState.RepositoryCount {
		return fmt.Errorf("url column validation failed: expected %d non-empty urls, got %d",
			preMigrationState.RepositoryCount, urlCount)
	}

	// Validate that 'location' column no longer exists (check via raw SQL)
	rows, err := db.Query("PRAGMA table_info(repositories)")
	if err != nil {
		return fmt.Errorf("failed to get table info: %w", err)
	}
	defer rows.Close()

	hasLocation := false
	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("failed to scan table info: %w", err)
		}
		if name == "location" {
			hasLocation = true
		}
	}

	if hasLocation {
		return fmt.Errorf("'location' column still exists after migration")
	}

	return nil
}

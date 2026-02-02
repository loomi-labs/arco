package migrate_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/loomi-labs/arco/backend/ent"
	_ "github.com/loomi-labs/arco/backend/ent/runtime"
	"github.com/loomi-labs/arco/backend/util"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

// TestSchemaCompleteness ensures that all schema entities are properly validated in TestMigrationDataIntegrity.
// This test will fail when new schema files are added, forcing developers to update the migration test.
func TestSchemaCompleteness(t *testing.T) {
	// Known entities that are validated in TestMigrationDataIntegrity
	// Update this list when adding new entities to the schema
	knownEntities := []string{
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

	// Scan the schema directory for all .go files
	schemaDir := filepath.Join("..", "schema")
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		t.Fatalf("Failed to read schema directory: %v", err)
	}

	var foundEntities []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories like mixin/
		}
		if filepath.Ext(entry.Name()) == ".go" {
			foundEntities = append(foundEntities, entry.Name())
		}
	}

	// Sort both lists for comparison
	sort.Strings(knownEntities)
	sort.Strings(foundEntities)

	// Check for new entities
	var newEntities []string
	for _, found := range foundEntities {
		isKnown := false
		for _, known := range knownEntities {
			if found == known {
				isKnown = true
				break
			}
		}
		if !isKnown {
			newEntities = append(newEntities, found)
		}
	}

	// Check for removed entities
	var removedEntities []string
	for _, known := range knownEntities {
		isFound := false
		for _, found := range foundEntities {
			if known == found {
				isFound = true
				break
			}
		}
		if !isFound {
			removedEntities = append(removedEntities, known)
		}
	}

	// Report findings
	if len(newEntities) > 0 {
		t.Errorf(`New schema entities detected: %v

IMPORTANT: Please update TestMigrationDataIntegrity to validate these new entities:
1. Add data capture for the new entity in captureState()
2. Add validation logic for the new entity in validateMigration()
3. Add the entity to seed_data.sql if needed
4. Update the knownEntities list in TestSchemaCompleteness()

This ensures that future migrations properly preserve data for all entities.`, newEntities)
	}

	if len(removedEntities) > 0 {
		t.Errorf(`Schema entities removed: %v

Please update the knownEntities list in TestSchemaCompleteness() to remove these entities.`, removedEntities)
	}

	if len(newEntities) > 0 || len(removedEntities) > 0 {
		t.Logf("\nCurrent schema files: %v", foundEntities)
		t.Logf("Known entities list: %v", knownEntities)
	}
}

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
	BackupSchedules           []BackupScheduleRelation
	PruningRules              []PruningRuleRelation
	Repositories              []RepositoryData
	BackupProfiles            []BackupProfileData
	RepositoryCount           int
	BackupProfileCount        int
}

type ProfileRepositoryRelation struct {
	BackupProfileID int
	RepositoryID    int
}

type ArchiveRelation struct {
	ID           int
	Name         string
	Duration     float64
	BorgID       string
	WillBePruned bool
	RepositoryID int
	ProfileID    sql.NullInt64
}

type NotificationRelation struct {
	ID           int
	Message      string
	Type         string
	Seen         bool
	ProfileID    int
	RepositoryID int
}

type BackupScheduleRelation struct {
	ID        int
	Mode      string
	ProfileID int
}

type PruningRuleRelation struct {
	ID        int
	IsEnabled bool
	ProfileID int
}

type RepositoryData struct {
	ID                     int
	Name                   string
	Location               string // Note: renamed to 'url' in migration 20251217094826
	Password               string // Note: removed in migration 20251217094826
	StatsTotalChunks       int
	StatsTotalSize         int
	StatsTotalCsize        int
	StatsTotalUniqueChunks int
	StatsUniqueSize        int
	StatsUniqueCsize       int
}

type BackupProfileData struct {
	ID                       int
	Name                     string
	Prefix                   string
	BackupPaths              []string
	ExcludePaths             []string
	Icon                     string
	DataSectionCollapsed     bool
	ScheduleSectionCollapsed bool
}

// captureState captures the current state of the database
func captureState(db *sql.DB) (*MigrationState, error) {
	state := &MigrationState{}

	// Count repositories
	err := db.QueryRow("SELECT COUNT(*) FROM repositories").Scan(&state.RepositoryCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count repositories: %w", err)
	}

	// Capture repository data
	repoRows, err := db.Query(`SELECT id, name, location, password, stats_total_chunks, stats_total_size,
		stats_total_csize, stats_total_unique_chunks, stats_unique_size, stats_unique_csize
		FROM repositories ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("failed to query repositories: %w", err)
	}
	defer repoRows.Close()

	for repoRows.Next() {
		var repo RepositoryData
		if err := repoRows.Scan(&repo.ID, &repo.Name, &repo.Location, &repo.Password,
			&repo.StatsTotalChunks, &repo.StatsTotalSize, &repo.StatsTotalCsize,
			&repo.StatsTotalUniqueChunks, &repo.StatsUniqueSize, &repo.StatsUniqueCsize); err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}
		state.Repositories = append(state.Repositories, repo)
	}

	// Count backup profiles
	err = db.QueryRow("SELECT COUNT(*) FROM backup_profiles").Scan(&state.BackupProfileCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count backup profiles: %w", err)
	}

	// Capture backup profile data
	profileRows, err := db.Query(`SELECT id, name, prefix, backup_paths, exclude_paths, icon,
		data_section_collapsed, schedule_section_collapsed
		FROM backup_profiles ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("failed to query backup profiles: %w", err)
	}
	defer profileRows.Close()

	for profileRows.Next() {
		var profile BackupProfileData
		var backupPathsJSON, excludePathsJSON string
		if err := profileRows.Scan(&profile.ID, &profile.Name, &profile.Prefix, &backupPathsJSON,
			&excludePathsJSON, &profile.Icon, &profile.DataSectionCollapsed,
			&profile.ScheduleSectionCollapsed); err != nil {
			return nil, fmt.Errorf("failed to scan backup profile: %w", err)
		}

		// Parse JSON arrays
		if err := json.Unmarshal([]byte(backupPathsJSON), &profile.BackupPaths); err != nil {
			return nil, fmt.Errorf("failed to parse backup_paths JSON: %w", err)
		}
		if err := json.Unmarshal([]byte(excludePathsJSON), &profile.ExcludePaths); err != nil {
			return nil, fmt.Errorf("failed to parse exclude_paths JSON: %w", err)
		}

		state.BackupProfiles = append(state.BackupProfiles, profile)
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

	// Capture archive data
	rows, err = db.Query(`SELECT id, name, duration, borg_id, will_be_pruned,
		archive_repository, archive_backup_profile FROM archives ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("failed to query archives: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var archive ArchiveRelation
		if err := rows.Scan(&archive.ID, &archive.Name, &archive.Duration, &archive.BorgID,
			&archive.WillBePruned, &archive.RepositoryID, &archive.ProfileID); err != nil {
			return nil, fmt.Errorf("failed to scan archive: %w", err)
		}
		state.Archives = append(state.Archives, archive)
	}

	// Capture notification data
	rows, err = db.Query(`SELECT id, message, type, seen,
		notification_backup_profile, notification_repository FROM notifications ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var notif NotificationRelation
		if err := rows.Scan(&notif.ID, &notif.Message, &notif.Type, &notif.Seen,
			&notif.ProfileID, &notif.RepositoryID); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		state.Notifications = append(state.Notifications, notif)
	}

	// Capture backup schedule relationships
	rows, err = db.Query("SELECT id, mode, backup_profile_backup_schedule FROM backup_schedules ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("failed to query backup_schedules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schedule BackupScheduleRelation
		if err := rows.Scan(&schedule.ID, &schedule.Mode, &schedule.ProfileID); err != nil {
			return nil, fmt.Errorf("failed to scan backup_schedule: %w", err)
		}
		state.BackupSchedules = append(state.BackupSchedules, schedule)
	}

	// Capture pruning rule relationships
	rows, err = db.Query("SELECT id, is_enabled, backup_profile_pruning_rule FROM pruning_rules ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("failed to query pruning_rules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rule PruningRuleRelation
		if err := rows.Scan(&rule.ID, &rule.IsEnabled, &rule.ProfileID); err != nil {
			return nil, fmt.Errorf("failed to scan pruning_rule: %w", err)
		}
		state.PruningRules = append(state.PruningRules, rule)
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

	// Validate backup profiles and their data
	profiles, err := client.BackupProfile.Query().
		WithRepositories().
		Order(ent.Asc("id")).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query backup profiles with repositories: %w", err)
	}

	if len(profiles) != len(preMigrationState.BackupProfiles) {
		return fmt.Errorf("backup profiles count mismatch after detailed query: expected %d, got %d",
			len(preMigrationState.BackupProfiles), len(profiles))
	}

	// Build a map of original names to detect duplicates
	originalNameCounts := make(map[string]int)
	for _, profile := range preMigrationState.BackupProfiles {
		originalNameCounts[profile.Name]++
	}

	// Validate backup profile data
	for i, profile := range profiles {
		expected := preMigrationState.BackupProfiles[i]

		// For profiles that had duplicate names, verify the name was properly transformed
		if originalNameCounts[expected.Name] > 1 {
			// Find how many profiles with same name had lower IDs
			countWithLowerID := 0
			for _, p := range preMigrationState.BackupProfiles {
				if p.Name == expected.Name && p.ID < expected.ID {
					countWithLowerID++
				}
			}

			if countWithLowerID == 0 {
				// First occurrence keeps original name
				if profile.Name != expected.Name {
					return fmt.Errorf("backup profile %d: first occurrence should keep original name %q, got %q",
						profile.ID, expected.Name, profile.Name)
				}
			} else {
				// Subsequent occurrences get " (N)" appended
				expectedNewName := fmt.Sprintf("%s (%d)", expected.Name, countWithLowerID)
				if profile.Name != expectedNewName {
					return fmt.Errorf("backup profile %d: duplicate name should be renamed to %q, got %q",
						profile.ID, expectedNewName, profile.Name)
				}
			}
		} else if profile.Name != expected.Name {
			return fmt.Errorf("backup profile %d: name mismatch: expected %q, got %q",
				profile.ID, expected.Name, profile.Name)
		}

		if profile.Prefix != expected.Prefix {
			return fmt.Errorf("backup profile %d: prefix mismatch: expected %q, got %q",
				profile.ID, expected.Prefix, profile.Prefix)
		}

		if !reflect.DeepEqual(profile.BackupPaths, expected.BackupPaths) {
			return fmt.Errorf("backup profile %d: backup_paths mismatch: expected %v, got %v",
				profile.ID, expected.BackupPaths, profile.BackupPaths)
		}

		if !reflect.DeepEqual(profile.ExcludePaths, expected.ExcludePaths) {
			return fmt.Errorf("backup profile %d: exclude_paths mismatch: expected %v, got %v",
				profile.ID, expected.ExcludePaths, profile.ExcludePaths)
		}

		if string(profile.Icon) != expected.Icon {
			return fmt.Errorf("backup profile %d: icon mismatch: expected %q, got %q",
				profile.ID, expected.Icon, profile.Icon)
		}

		if profile.DataSectionCollapsed != expected.DataSectionCollapsed {
			return fmt.Errorf("backup profile %d: data_section_collapsed mismatch: expected %t, got %t",
				profile.ID, expected.DataSectionCollapsed, profile.DataSectionCollapsed)
		}

		if profile.ScheduleSectionCollapsed != expected.ScheduleSectionCollapsed {
			return fmt.Errorf("backup profile %d: schedule_section_collapsed mismatch: expected %t, got %t",
				profile.ID, expected.ScheduleSectionCollapsed, profile.ScheduleSectionCollapsed)
		}

		// Validate new fields have proper defaults
		if string(profile.CompressionMode) != "lz4" {
			return fmt.Errorf("backup profile %d: compression_mode should default to 'lz4', got %q",
				profile.ID, profile.CompressionMode)
		}

		if profile.AdvancedSectionCollapsed != true {
			return fmt.Errorf("backup profile %d: advanced_section_collapsed should default to true, got %t",
				profile.ID, profile.AdvancedSectionCollapsed)
		}
	}

	// Validate that all backup profile names are unique after migration
	postMigrationNames := make(map[string]int)
	for _, profile := range profiles {
		if existingID, exists := postMigrationNames[profile.Name]; exists {
			return fmt.Errorf("backup profile names not unique after migration: profiles %d and %d both have name %q",
				existingID, profile.ID, profile.Name)
		}
		postMigrationNames[profile.Name] = profile.ID
	}

	// Validate backup_profile_repositories relationships using Ent edges
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

		// Validate basic fields
		if archive.Name != expected.Name {
			return fmt.Errorf("archive %d: name mismatch: expected %q, got %q",
				archive.ID, expected.Name, archive.Name)
		}

		if archive.Duration != expected.Duration {
			return fmt.Errorf("archive %d: duration mismatch: expected %f, got %f",
				archive.ID, expected.Duration, archive.Duration)
		}

		if archive.BorgID != expected.BorgID {
			return fmt.Errorf("archive %d: borg_id mismatch: expected %q, got %q",
				archive.ID, expected.BorgID, archive.BorgID)
		}

		if archive.WillBePruned != expected.WillBePruned {
			return fmt.Errorf("archive %d: will_be_pruned mismatch: expected %t, got %t",
				archive.ID, expected.WillBePruned, archive.WillBePruned)
		}

		// Validate repository relationship
		archiveRepoID, err := archive.QueryRepository().OnlyID(ctx)
		if err != nil {
			return fmt.Errorf("failed to get repository ID for archive %d: %w", archive.ID, err)
		}

		if archiveRepoID != expected.RepositoryID {
			return fmt.Errorf("archive %d: repository relationship mismatch: expected %d, got %d",
				archive.ID, expected.RepositoryID, archiveRepoID)
		}

		// Validate backup profile relationship
		if expected.ProfileID.Valid {
			profileID, err := archive.QueryBackupProfile().OnlyID(ctx)
			if err != nil {
				return fmt.Errorf("failed to get backup profile ID for archive %d: %w", archive.ID, err)
			}

			if profileID != int(expected.ProfileID.Int64) {
				return fmt.Errorf("archive %d: backup profile relationship mismatch: expected %d, got %d",
					archive.ID, expected.ProfileID.Int64, profileID)
			}
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

		// Validate basic fields
		if notif.Message != expected.Message {
			return fmt.Errorf("notification %d: message mismatch: expected %q, got %q",
				notif.ID, expected.Message, notif.Message)
		}

		if string(notif.Type) != expected.Type {
			return fmt.Errorf("notification %d: type mismatch: expected %q, got %q",
				notif.ID, expected.Type, notif.Type)
		}

		if notif.Seen != expected.Seen {
			return fmt.Errorf("notification %d: seen mismatch: expected %t, got %t",
				notif.ID, expected.Seen, notif.Seen)
		}

		// Validate relationships
		profileID, err := notif.QueryBackupProfile().OnlyID(ctx)
		if err != nil {
			return fmt.Errorf("failed to get backup profile ID for notification %d: %w", notif.ID, err)
		}

		if profileID != expected.ProfileID {
			return fmt.Errorf("notification %d: backup profile relationship mismatch: expected %d, got %d",
				notif.ID, expected.ProfileID, profileID)
		}

		repoID, err := notif.QueryRepository().OnlyID(ctx)
		if err != nil {
			return fmt.Errorf("failed to get repository ID for notification %d: %w", notif.ID, err)
		}

		if repoID != expected.RepositoryID {
			return fmt.Errorf("notification %d: repository relationship mismatch: expected %d, got %d",
				notif.ID, expected.RepositoryID, repoID)
		}
	}

	// Validate repository data and transformation from 'location' to 'url'
	repos, err := client.Repository.Query().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query repositories: %w", err)
	}

	if len(repos) != len(preMigrationState.Repositories) {
		return fmt.Errorf("repository count mismatch after detailed query: expected %d, got %d",
			len(preMigrationState.Repositories), len(repos))
	}

	for i, repo := range repos {
		expected := preMigrationState.Repositories[i]

		// Validate URL field was populated from location field
		if repo.URL != expected.Location {
			return fmt.Errorf("repository %d: url mismatch: expected %q (from location), got %q",
				repo.ID, expected.Location, repo.URL)
		}

		// Validate basic fields
		if repo.Name != expected.Name {
			return fmt.Errorf("repository %d: name mismatch: expected %q, got %q",
				repo.ID, expected.Name, repo.Name)
		}

		// Validate hasPassword field was set based on whether password was non-empty
		expectedHasPassword := expected.Password != ""
		if repo.HasPassword != expectedHasPassword {
			return fmt.Errorf("repository %d: hasPassword mismatch: expected %v (password was %q), got %v",
				repo.ID, expectedHasPassword, expected.Password, repo.HasPassword)
		}

		// Validate stats fields
		if repo.StatsTotalChunks != expected.StatsTotalChunks {
			return fmt.Errorf("repository %d: stats_total_chunks mismatch: expected %d, got %d",
				repo.ID, expected.StatsTotalChunks, repo.StatsTotalChunks)
		}

		if repo.StatsTotalSize != expected.StatsTotalSize {
			return fmt.Errorf("repository %d: stats_total_size mismatch: expected %d, got %d",
				repo.ID, expected.StatsTotalSize, repo.StatsTotalSize)
		}

		if repo.StatsTotalCsize != expected.StatsTotalCsize {
			return fmt.Errorf("repository %d: stats_total_csize mismatch: expected %d, got %d",
				repo.ID, expected.StatsTotalCsize, repo.StatsTotalCsize)
		}

		if repo.StatsTotalUniqueChunks != expected.StatsTotalUniqueChunks {
			return fmt.Errorf("repository %d: stats_total_unique_chunks mismatch: expected %d, got %d",
				repo.ID, expected.StatsTotalUniqueChunks, repo.StatsTotalUniqueChunks)
		}

		if repo.StatsUniqueSize != expected.StatsUniqueSize {
			return fmt.Errorf("repository %d: stats_unique_size mismatch: expected %d, got %d",
				repo.ID, expected.StatsUniqueSize, repo.StatsUniqueSize)
		}

		if repo.StatsUniqueCsize != expected.StatsUniqueCsize {
			return fmt.Errorf("repository %d: stats_unique_csize mismatch: expected %d, got %d",
				repo.ID, expected.StatsUniqueCsize, repo.StatsUniqueCsize)
		}
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

	// Validate backup schedules
	schedules, err := client.BackupSchedule.Query().
		Order(ent.Asc("id")).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query backup schedules: %w", err)
	}

	if len(schedules) != len(preMigrationState.BackupSchedules) {
		return fmt.Errorf("backup_schedules count mismatch: expected %d, got %d",
			len(preMigrationState.BackupSchedules), len(schedules))
	}

	for i, schedule := range schedules {
		expected := preMigrationState.BackupSchedules[i]
		profileID, err := schedule.QueryBackupProfile().OnlyID(ctx)
		if err != nil {
			return fmt.Errorf("failed to get backup profile ID for schedule %d: %w", schedule.ID, err)
		}

		// Determine expected mode after migration ('hourly' -> 'minute_interval')
		expectedMode := expected.Mode
		if expectedMode == "hourly" {
			expectedMode = "minute_interval"
		}

		if schedule.ID != expected.ID || profileID != expected.ProfileID || string(schedule.Mode) != expectedMode {
			return fmt.Errorf("backup_schedule relationship mismatch at index %d: expected (id=%d, profile=%d, mode=%s), got (id=%d, profile=%d, mode=%s)",
				i, expected.ID, expected.ProfileID, expectedMode, schedule.ID, profileID, schedule.Mode)
		}

		// Validate interval_minutes has default value of 60
		if schedule.IntervalMinutes != 60 {
			return fmt.Errorf("backup_schedule %d: interval_minutes should default to 60, got %d",
				schedule.ID, schedule.IntervalMinutes)
		}
	}

	// Validate pruning rules
	pruningRules, err := client.PruningRule.Query().
		Order(ent.Asc("id")).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query pruning rules: %w", err)
	}

	if len(pruningRules) != len(preMigrationState.PruningRules) {
		return fmt.Errorf("pruning_rules count mismatch: expected %d, got %d",
			len(preMigrationState.PruningRules), len(pruningRules))
	}

	for i, rule := range pruningRules {
		expected := preMigrationState.PruningRules[i]
		profileID, err := rule.QueryBackupProfile().OnlyID(ctx)
		if err != nil {
			return fmt.Errorf("failed to get backup profile ID for pruning rule %d: %w", rule.ID, err)
		}

		if rule.ID != expected.ID || profileID != expected.ProfileID || rule.IsEnabled != expected.IsEnabled {
			return fmt.Errorf("pruning_rule relationship mismatch at index %d: expected (id=%d, profile=%d, enabled=%t), got (id=%d, profile=%d, enabled=%t)",
				i, expected.ID, expected.ProfileID, expected.IsEnabled, rule.ID, profileID, rule.IsEnabled)
		}
	}

	return nil
}

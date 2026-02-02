package backup_profile

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/archive"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/backupschedule"
	"github.com/loomi-labs/arco/backend/ent/notification"
	"github.com/loomi-labs/arco/backend/ent/repository"
	"github.com/loomi-labs/arco/backend/ent/schema"
	"github.com/loomi-labs/arco/backend/util"
	"github.com/negrel/assert"
	"github.com/wailsapp/wails/v3/pkg/application"
	"go.uber.org/zap"
)

// Service contains the business logic for backup profiles
type Service struct {
	log                      *zap.SugaredLogger
	db                       *ent.Client
	state                    *state.State
	config                   *types.Config
	eventEmitter             types.EventEmitter
	backupScheduleChangedCh  chan struct{}
	pruningScheduleChangedCh chan struct{}
	repositoryService        RepositoryServiceInterface
	ctx                      context.Context
}

// RepositoryServiceInterface defines the methods needed from repository service
type RepositoryServiceInterface interface {
	QueueBackup(ctx context.Context, backupId types.BackupId) (string, error)
	QueuePrune(ctx context.Context, backupId types.BackupId) (string, error)
	QueueArchiveDelete(ctx context.Context, archiveId int) (string, error)
}

// ServiceInternal provides backend-only methods that should not be exposed to frontend
type ServiceInternal struct {
	*Service
}

// NewService creates a new backup profile service
func NewService(log *zap.SugaredLogger, state *state.State, config *types.Config) *ServiceInternal {
	return &ServiceInternal{
		Service: &Service{
			log:    log,
			state:  state,
			config: config,
		},
	}
}

// Init initializes the service with remaining dependencies
func (si *ServiceInternal) Init(ctx context.Context, db *ent.Client, eventEmitter types.EventEmitter, backupScheduleChangedCh, pruningScheduleChangedCh chan struct{}, repositoryService RepositoryServiceInterface) {
	si.ctx = ctx
	si.db = db
	si.eventEmitter = eventEmitter
	si.backupScheduleChangedCh = backupScheduleChangedCh
	si.pruningScheduleChangedCh = pruningScheduleChangedCh
	si.repositoryService = repositoryService
}

// mustHaveDB panics if db is nil. This is a programming error guard.
func (s *Service) mustHaveDB() {
	if s.db == nil {
		panic("BackupProfileService: database client is nil")
	}
}

/***********************************/
/********** Backup Profile *********/
/***********************************/

// getBackupProfileEnt is an internal method that returns the raw ent.BackupProfile
func (s *Service) getBackupProfileEnt(ctx context.Context, id int) (*ent.BackupProfile, error) {
	s.mustHaveDB()
	return s.db.BackupProfile.
		Query().
		WithRepositories().
		WithBackupSchedule().
		WithPruningRule().
		Where(backupprofile.ID(id)).
		Only(ctx)
}

func (s *Service) GetBackupProfile(ctx context.Context, id int) (*BackupProfile, error) {
	entProfile, err := s.getBackupProfileEnt(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toBackupProfile(ctx, entProfile)
}

func (s *Service) GetBackupProfiles(ctx context.Context) ([]*BackupProfile, error) {
	s.mustHaveDB()
	entProfiles, err := s.db.BackupProfile.
		Query().
		WithRepositories().
		WithBackupSchedule().
		WithPruningRule().
		Order(func(sel *sql.Selector) {
			// Order by name, case-insensitive
			sel.OrderExpr(sql.Expr(fmt.Sprintf("%s COLLATE NOCASE", backupprofile.FieldName)))
		}).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return s.toBackupProfiles(ctx, entProfiles)
}

type BackupProfileFilter struct {
	Id              int    `json:"id,omitempty"`
	Name            string `json:"name"`
	IsAllFilter     bool   `json:"isAllFilter"`
	IsUnknownFilter bool   `json:"isUnknownFilter"`
}

func (s *Service) GetBackupProfileFilterOptions(ctx context.Context, repoId int) ([]BackupProfileFilter, error) {
	s.mustHaveDB()
	profiles, err := s.db.BackupProfile.
		Query().
		Where(backupprofile.HasRepositoriesWith(repository.ID(repoId))).
		Order(ent.Desc(backupprofile.FieldName)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	filters := make([]BackupProfileFilter, len(profiles))
	for i, p := range profiles {
		filters[i] = BackupProfileFilter{Id: p.ID, Name: p.Name}
	}

	hasForeignArchives, err := s.db.Repository.
		Query().
		Where(repository.And(
			repository.ID(repoId),
			repository.HasArchivesWith(archive.Not(archive.HasBackupProfile())),
		)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}

	if hasForeignArchives {
		filters = append([]BackupProfileFilter{{Name: "Not defined", IsUnknownFilter: true}}, filters...)
	}

	if len(filters) > 1 {
		filters = append([]BackupProfileFilter{{Name: "All", IsAllFilter: true}}, filters...)
	}

	return filters, nil
}

func (s *Service) NewBackupProfile(ctx context.Context) (*BackupProfile, error) {
	s.mustHaveDB()
	// Choose the first icon that is not already in use
	all, err := s.db.BackupProfile.
		Query().
		Select(backupprofile.FieldIcon).
		All(ctx)
	if err != nil {
		return nil, err
	}
	icons := make(map[backupprofile.Icon]bool)
	for _, p := range all {
		icons[p.Icon] = true
	}
	selectedIcon := backupprofile.IconHome
	for _, icon := range types.AllIcons {
		if !icons[icon] {
			selectedIcon = icon
			break
		}
	}

	// We only care about the hour, minute, second and nanosecond (in local time for display purpose)
	firstDayOfMonthAtNine := time.Date(time.Now().Year(), 1, 1, 9, 0, 0, 0, time.Local)
	schedule := &BackupSchedule{
		Mode:      backupschedule.ModeMinuteInterval,
		DailyAt:   firstDayOfMonthAtNine,
		Weekday:   backupschedule.WeekdayMonday,
		WeeklyAt:  firstDayOfMonthAtNine,
		Monthday:  1,
		MonthlyAt: firstDayOfMonthAtNine,
	}

	pruningRule := &PruningRule{
		IsEnabled:      false,
		KeepHourly:     defaultPruningOption.KeepHourly,
		KeepDaily:      defaultPruningOption.KeepDaily,
		KeepWeekly:     defaultPruningOption.KeepWeekly,
		KeepMonthly:    defaultPruningOption.KeepMonthly,
		KeepYearly:     defaultPruningOption.KeepYearly,
		KeepWithinDays: 30,
	}

	return &BackupProfile{
		ID:                       0,
		Name:                     "",
		Prefix:                   "",
		BackupPaths:              make([]string, 0),
		ExcludePaths:             make([]string, 0),
		ExcludeCaches:            true,
		Icon:                     selectedIcon,
		CompressionMode:          backupprofile.CompressionModeLz4, // Default compression
		CompressionLevel:         nil,                              // lz4 doesn't use levels
		AdvancedSectionCollapsed: true,                             // Collapsed by default
		Repositories:             make([]RepositorySummary, 0),
		BackupSchedule:           schedule,
		PruningRule:              pruningRule,
		ArchiveCount:             0,
	}, nil
}

func (s *Service) GetDirectorySuggestions() []string {
	home, _ := os.UserHomeDir()
	if home != "" {
		return []string{home}
	}
	return []string{}
}

func (s *Service) DoesPathExist(path string) bool {
	_, err := os.Stat(util.ExpandPath(path))
	return err == nil
}

func (s *Service) IsDirectory(path string) bool {
	info, err := os.Stat(util.ExpandPath(path))
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (s *Service) IsDirectoryEmpty(path string) bool {
	path = util.ExpandPath(path)
	if !s.IsDirectory(path) {
		return false
	}

	f, err := os.Open(path)
	if err != nil {
		return false
	}
	//goland:noinspection GoUnhandledErrorResult
	defer f.Close()

	_, err = f.Readdirnames(1)
	return err != nil
}

func (s *Service) CreateDirectory(path string) error {
	s.log.Debug(fmt.Sprintf("Creating directory %s", path))
	return os.MkdirAll(util.ExpandPath(path), 0755)
}

func (s *Service) GetPrefixSuggestion(ctx context.Context, name string) (string, error) {
	s.mustHaveDB()
	if name == "" {
		return "", errors.New("name cannot be empty")
	}
	name = strings.ToLower(name)

	// Remove all non-alphanumeric characters
	re := regexp.MustCompile("[^a-z0-9]")
	prefix := re.ReplaceAllString(name, "")

	if prefix == "" {
		return "", errors.New("name must contain at least one alphanumeric character")
	}

	fullPrefix := prefix + "-"

	exist, err := s.db.BackupProfile.
		Query().
		Where(backupprofile.Prefix(fullPrefix)).
		Exist(ctx)
	if err != nil {
		return "", err
	}
	if exist {
		// If the prefix already exists, we create a new one by appending a random number
		prefix = fmt.Sprintf("%s%04d", prefix, rand.Intn(1000))
		return s.GetPrefixSuggestion(ctx, prefix)
	}
	return fullPrefix, nil
}

// ValidateBackupProfileName validates a backup profile name and returns an error message if invalid.
// Returns empty string if valid.
func (s *Service) ValidateBackupProfileName(ctx context.Context, name string) (string, error) {
	s.mustHaveDB()
	if name == "" {
		return "Name is required", nil
	}
	if len(name) < schema.ValBackupProfileMinNameLength {
		return fmt.Sprintf("Name must be at least %d characters long", schema.ValBackupProfileMinNameLength), nil
	}
	if len(name) > schema.ValBackupProfileMaxNameLength {
		return fmt.Sprintf("Name can not be longer than %d characters", schema.ValBackupProfileMaxNameLength), nil
	}

	exist, err := s.db.BackupProfile.
		Query().
		Where(backupprofile.Name(name)).
		Exist(ctx)
	if err != nil {
		return "", err
	}
	if exist {
		return "Backup profile name must be unique", nil
	}

	return "", nil
}

func (s *Service) CreateBackupProfile(ctx context.Context, backup BackupProfile, repositoryIds []int) (*BackupProfile, error) {
	s.mustHaveDB()
	s.log.Debug(fmt.Sprintf("Creating backup profile %d", backup.ID))

	// Validate compression settings
	if err := validateCompression(backup.CompressionMode, backup.CompressionLevel); err != nil {
		return nil, fmt.Errorf("invalid compression settings: %w", err)
	}

	profile, err := s.db.BackupProfile.
		Create().
		SetName(backup.Name).
		SetPrefix(backup.Prefix).
		SetBackupPaths(backup.BackupPaths).
		SetExcludePaths(backup.ExcludePaths).
		SetExcludeCaches(backup.ExcludeCaches).
		SetIcon(backup.Icon).
		SetCompressionMode(backup.CompressionMode).
		SetNillableCompressionLevel(backup.CompressionLevel).
		SetAdvancedSectionCollapsed(backup.AdvancedSectionCollapsed).
		AddRepositoryIDs(repositoryIds...).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	s.eventEmitter.EmitEvent(ctx, types.EventBackupProfileCreatedString())
	return s.toBackupProfile(ctx, profile)
}

func (s *Service) UpdateBackupProfile(ctx context.Context, backup BackupProfile) error {
	s.mustHaveDB()
	s.log.Debug(fmt.Sprintf("Updating backup profile %d", backup.ID))

	// Auto-clear compression level for modes that don't support it
	if backup.CompressionMode == backupprofile.CompressionModeNone ||
		backup.CompressionMode == backupprofile.CompressionModeLz4 {
		backup.CompressionLevel = nil
	}

	// Validate compression settings
	if err := validateCompression(backup.CompressionMode, backup.CompressionLevel); err != nil {
		return fmt.Errorf("invalid compression settings: %w", err)
	}

	update := s.db.BackupProfile.
		UpdateOneID(backup.ID).
		SetName(backup.Name).
		SetIcon(backup.Icon).
		SetBackupPaths(backup.BackupPaths).
		SetExcludePaths(backup.ExcludePaths).
		SetExcludeCaches(backup.ExcludeCaches).
		SetDataSectionCollapsed(backup.DataSectionCollapsed).
		SetScheduleSectionCollapsed(backup.ScheduleSectionCollapsed).
		SetCompressionMode(backup.CompressionMode).
		SetAdvancedSectionCollapsed(backup.AdvancedSectionCollapsed)

	// Use ClearCompressionLevel for modes that don't support it (SetNillableCompressionLevel(nil) is a no-op)
	if backup.CompressionMode == backupprofile.CompressionModeNone ||
		backup.CompressionMode == backupprofile.CompressionModeLz4 {
		update = update.ClearCompressionLevel()
	} else {
		update = update.SetNillableCompressionLevel(backup.CompressionLevel)
	}

	err := update.Exec(ctx)
	if err != nil {
		return err
	}
	s.eventEmitter.EmitEvent(ctx, types.EventBackupProfileUpdatedString())
	return nil
}

// DeleteBackupProfile deletes a backup profile and optionally its archives
func (s *Service) DeleteBackupProfile(ctx context.Context, backupProfileId int, deleteArchives bool) error {
	s.mustHaveDB()
	backupProfile, err := s.GetBackupProfile(ctx, backupProfileId)
	if err != nil {
		return err
	}

	// If deleteArchives is true, queue archive deletions for each repository
	if deleteArchives && s.repositoryService != nil {
		for _, repo := range backupProfile.Repositories {
			// Query archives for this backup profile and repository
			archives, err := s.db.Archive.Query().
				Where(
					archive.HasRepositoryWith(repository.ID(repo.ID)),
					archive.HasBackupProfileWith(backupprofile.ID(backupProfileId)),
				).
				All(ctx)
			if err != nil {
				s.log.Errorw("Failed to query archives for deletion",
					"backupProfileId", backupProfileId,
					"repositoryId", repo.ID,
					"error", err)
				continue
			}

			// Queue delete operation for each archive
			for _, arch := range archives {
				_, err := s.repositoryService.QueueArchiveDelete(ctx, arch.ID)
				if err != nil {
					s.log.Errorw("Failed to queue archive delete",
						"archiveId", arch.ID,
						"error", err)
				}
			}
		}
	}

	err = s.db.BackupProfile.
		DeleteOneID(backupProfileId).
		Exec(ctx)
	if err != nil {
		return err
	}
	s.eventEmitter.EmitEvent(ctx, types.EventBackupProfileDeleted.String())

	return nil
}

func (s *Service) AddRepositoryToBackupProfile(ctx context.Context, backupProfileId int, repositoryId int) error {
	s.mustHaveDB()
	bs, err := s.GetBackupProfile(ctx, backupProfileId)
	if err != nil {
		return err
	}
	assert.NotNil(bs.Repositories, "backup profile does not have repositories")
	for _, r := range bs.Repositories {
		if r.ID == repositoryId {
			return fmt.Errorf("repository is already in the backup profile")
		}
	}
	err = s.db.BackupProfile.
		UpdateOneID(backupProfileId).
		AddRepositoryIDs(repositoryId).
		Exec(ctx)
	if err != nil {
		return err
	}
	s.eventEmitter.EmitEvent(ctx, types.EventBackupProfileUpdatedString())
	return nil
}

// RemoveRepositoryFromBackupProfile removes a repository from a backup profile
func (s *Service) RemoveRepositoryFromBackupProfile(ctx context.Context, backupProfileId int, repositoryId int, deleteArchives bool) error {
	s.mustHaveDB()
	s.log.Debug(fmt.Sprintf("Removing repository %d from backup profile %d", repositoryId, backupProfileId))

	// Get the backup profile with the repository
	backupProfile, err := s.db.BackupProfile.
		Query().
		Where(backupprofile.And(
			backupprofile.ID(backupProfileId),
			backupprofile.HasRepositoriesWith(repository.ID(repositoryId)),
		)).
		WithRepositories(func(q *ent.RepositoryQuery) {
			q.Where(repository.ID(repositoryId))
		}).
		Only(ctx)
	if err != nil {
		return err
	}

	// Check if the repository is the only one in the backup profile
	cnt, err := s.db.Repository.
		Query().
		Where(repository.HasBackupProfilesWith(backupprofile.ID(backupProfileId))).
		Count(ctx)
	if err != nil {
		return err
	}
	if cnt <= 1 {
		assert.NotEqual(cnt, 0, "backup profile does not have repositories")
		return fmt.Errorf("cannot remove the only repository from the backup profile")
	}

	// If deleteArchives is true, queue archive deletions for this repository
	if deleteArchives && len(backupProfile.Edges.Repositories) > 0 && s.repositoryService != nil {
		repo := backupProfile.Edges.Repositories[0]
		if repo.ID == repositoryId {
			// Query archives for this backup profile and repository
			archives, err := s.db.Archive.Query().
				Where(
					archive.HasRepositoryWith(repository.ID(repositoryId)),
					archive.HasBackupProfileWith(backupprofile.ID(backupProfileId)),
				).
				All(ctx)
			if err != nil {
				s.log.Errorw("Failed to query archives for deletion",
					"backupProfileId", backupProfileId,
					"repositoryId", repositoryId,
					"error", err)
			} else {
				// Queue delete operation for each archive
				for _, arch := range archives {
					_, err := s.repositoryService.QueueArchiveDelete(ctx, arch.ID)
					if err != nil {
						s.log.Errorw("Failed to queue archive delete",
							"archiveId", arch.ID,
							"error", err)
					}
				}
			}
		}
	}

	err = s.db.BackupProfile.
		UpdateOneID(backupProfileId).
		RemoveRepositoryIDs(repositoryId).
		Exec(ctx)
	if err != nil {
		return err
	}
	s.eventEmitter.EmitEvent(ctx, types.EventBackupProfileUpdatedString())
	return nil
}

type SelectDirectoryData struct {
	Title      string `json:"title"`
	Message    string `json:"message"`
	ButtonText string `json:"buttonText"`
}

func (s *Service) SelectDirectory(data SelectDirectoryData) (string, error) {
	dialog := application.Get().Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		CanChooseDirectories:            true,
		CanChooseFiles:                  false,
		CanCreateDirectories:            true,
		ShowHiddenFiles:                 false,
		ResolvesAliases:                 true,
		AllowsMultipleSelection:         false,
		HideExtension:                   true,
		CanSelectHiddenExtension:        false,
		TreatsFilePackagesAsDirectories: false,
		AllowsOtherFileTypes:            false,
		Filters:                         nil,
		Window:                          nil,
		Title:                           data.Title,
		Message:                         data.Message,
		ButtonText:                      data.ButtonText,
		Directory:                       "",
	})
	return dialog.PromptForSingleSelection()
}

/***********************************/
/********** Backup Schedule ********/
/***********************************/

// applyScheduleDefaults ensures all schedule fields have valid values.
// This prevents validation errors when fields are empty/zero.
func applyScheduleDefaultsEnt(schedule *ent.BackupSchedule) {
	defaultTime := time.Date(time.Now().Year(), 1, 1, 9, 0, 0, 0, time.Local)

	if schedule.IntervalMinutes == 0 {
		schedule.IntervalMinutes = 60
	}
	if schedule.Weekday == "" {
		schedule.Weekday = backupschedule.WeekdayMonday
	}
	if schedule.Monthday == 0 {
		schedule.Monthday = 1
	}
	if schedule.DailyAt.IsZero() {
		schedule.DailyAt = defaultTime
	}
	if schedule.WeeklyAt.IsZero() {
		schedule.WeeklyAt = defaultTime
	}
	if schedule.MonthlyAt.IsZero() {
		schedule.MonthlyAt = defaultTime
	}
}

func (s *Service) SaveBackupSchedule(ctx context.Context, backupProfileId int, schedule BackupSchedule) error {
	s.mustHaveDB()
	s.log.Debug(fmt.Sprintf("Saving backup schedule for backup profile %d", backupProfileId))

	// Convert to ent type for internal operations
	entSchedule := schedule.ToEnt()

	// Apply defaults to ensure all fields have valid values
	applyScheduleDefaultsEnt(entSchedule)

	defer s.sendBackupScheduleChanged()
	doesExist, err := s.db.BackupSchedule.
		Query().
		Where(backupschedule.HasBackupProfileWith(backupprofile.ID(backupProfileId))).
		Exist(ctx)
	if err != nil {
		return err
	}

	var nextRun *time.Time
	if entSchedule.Mode != backupschedule.ModeDisabled {
		nr, err := getNextBackupTime(entSchedule, time.Now())
		if err != nil {
			return err
		}
		nextRun = &nr
		s.log.Debugf("Next run in %s", nextRun.Sub(time.Now()))
	}

	if doesExist {
		return s.db.BackupSchedule.
			Update().
			Where(backupschedule.HasBackupProfileWith(backupprofile.ID(backupProfileId))).
			SetMode(entSchedule.Mode).
			SetIntervalMinutes(entSchedule.IntervalMinutes).
			SetDailyAt(entSchedule.DailyAt).
			SetWeeklyAt(entSchedule.WeeklyAt).
			SetWeekday(entSchedule.Weekday).
			SetMonthlyAt(entSchedule.MonthlyAt).
			SetMonthday(entSchedule.Monthday).
			ClearNextRun().
			SetNillableNextRun(nextRun).
			Exec(ctx)
	}
	return s.db.BackupSchedule.
		Create().
		SetMode(entSchedule.Mode).
		SetIntervalMinutes(entSchedule.IntervalMinutes).
		SetDailyAt(entSchedule.DailyAt).
		SetWeeklyAt(entSchedule.WeeklyAt).
		SetWeekday(entSchedule.Weekday).
		SetMonthlyAt(entSchedule.MonthlyAt).
		SetMonthday(entSchedule.Monthday).
		SetNillableNextRun(nextRun).
		SetBackupProfileID(backupProfileId).
		Exec(ctx)
}

func (s *Service) sendBackupScheduleChanged() {
	if s.backupScheduleChangedCh == nil {
		return
	}
	s.backupScheduleChangedCh <- struct{}{}
}

/***********************************/
/********** Pruning Options ********/
/***********************************/

type PruningOptionName string

const (
	PruningOptionNone   PruningOptionName = "none"
	PruningOptionFew    PruningOptionName = "few"
	PruningOptionMany   PruningOptionName = "many"
	PruningOptionCustom PruningOptionName = "custom"
)

func (p PruningOptionName) String() string {
	return string(p)
}

type PruningOption struct {
	Name        PruningOptionName `json:"name"`
	KeepHourly  int               `json:"keepHourly"`
	KeepDaily   int               `json:"keepDaily"`
	KeepWeekly  int               `json:"keepWeekly"`
	KeepMonthly int               `json:"keepMonthly"`
	KeepYearly  int               `json:"keepYearly"`
}

var PruningOptions = []PruningOption{
	{Name: PruningOptionNone},
	{Name: PruningOptionFew, KeepHourly: 8, KeepDaily: 7, KeepWeekly: 4, KeepMonthly: 3, KeepYearly: 1},
	{Name: PruningOptionMany, KeepHourly: 24, KeepDaily: 14, KeepWeekly: 8, KeepMonthly: 6, KeepYearly: 2},
	{Name: PruningOptionCustom},
}

var defaultPruningOption = PruningOptions[2]

type GetPruningOptionsResponse struct {
	Options []PruningOption `json:"options"`
}

func (s *Service) GetPruningOptions() GetPruningOptionsResponse {
	return GetPruningOptionsResponse{Options: PruningOptions}
}

func (s *Service) SavePruningRule(ctx context.Context, backupId int, rule PruningRule) (*PruningRule, error) {
	s.mustHaveDB()
	defer s.sendPruningRuleChanged()

	backupProfile, err := s.GetBackupProfile(ctx, backupId)
	if err != nil {
		return nil, err
	}

	// Convert custom BackupSchedule to ent for getNextPruneTime
	var entSchedule *ent.BackupSchedule
	if backupProfile.BackupSchedule != nil {
		entSchedule = backupProfile.BackupSchedule.ToEnt()
	}
	nextRun := getNextPruneTime(entSchedule, time.Now())

	var savedRule *ent.PruningRule
	if backupProfile.PruningRule != nil {
		s.log.Debug(fmt.Sprintf("Updating pruning rule %d for backup profile %d", rule.ID, backupId))
		savedRule, err = s.db.PruningRule.
			// We ignore the ID from the given rule and get it from the db directly
			UpdateOneID(backupProfile.PruningRule.ID).
			SetIsEnabled(rule.IsEnabled).
			SetKeepHourly(rule.KeepHourly).
			SetKeepDaily(rule.KeepDaily).
			SetKeepWeekly(rule.KeepWeekly).
			SetKeepMonthly(rule.KeepMonthly).
			SetKeepYearly(rule.KeepYearly).
			SetKeepWithinDays(rule.KeepWithinDays).
			SetNextRun(nextRun).
			Save(ctx)
	} else {
		s.log.Debug(fmt.Sprintf("Creating pruning rule for backup profile %d", backupId))
		savedRule, err = s.db.PruningRule.
			Create().
			SetIsEnabled(rule.IsEnabled).
			SetKeepHourly(rule.KeepHourly).
			SetKeepDaily(rule.KeepDaily).
			SetKeepWeekly(rule.KeepWeekly).
			SetKeepMonthly(rule.KeepMonthly).
			SetKeepYearly(rule.KeepYearly).
			SetKeepWithinDays(rule.KeepWithinDays).
			SetBackupProfileID(backupId).
			SetNextRun(nextRun).
			Save(ctx)
	}
	if err != nil {
		return nil, err
	}
	return toPruningRule(savedRule), nil
}

func (s *Service) sendPruningRuleChanged() {
	s.log.Debug("Sending pruning rule changed event")
	if s.pruningScheduleChangedCh == nil {
		return
	}
	s.pruningScheduleChangedCh <- struct{}{}
}

/***********************************/
/********** Custom Types ***********/
/***********************************/

// RepositorySummary contains minimal repository info for BackupProfile display
type RepositorySummary struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// BackupSchedule is a standalone view of ent.BackupSchedule without back-edges
type BackupSchedule struct {
	ID              int                    `json:"id"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
	Mode            backupschedule.Mode    `json:"mode"`
	IntervalMinutes uint16                 `json:"intervalMinutes"`
	DailyAt         time.Time              `json:"dailyAt"`
	Weekday         backupschedule.Weekday `json:"weekday"`
	WeeklyAt        time.Time              `json:"weeklyAt"`
	Monthday        uint8                  `json:"monthday"`
	MonthlyAt       time.Time              `json:"monthlyAt"`
	NextRun         time.Time              `json:"nextRun"`
	LastRun         *time.Time             `json:"lastRun"`
	LastRunStatus   *string                `json:"lastRunStatus"`
}

// PruningRule is a standalone view of ent.PruningRule without back-edges
type PruningRule struct {
	ID             int        `json:"id"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	IsEnabled      bool       `json:"isEnabled"`
	KeepHourly     int        `json:"keepHourly"`
	KeepDaily      int        `json:"keepDaily"`
	KeepWeekly     int        `json:"keepWeekly"`
	KeepMonthly    int        `json:"keepMonthly"`
	KeepYearly     int        `json:"keepYearly"`
	KeepWithinDays int        `json:"keepWithinDays"`
	NextRun        time.Time  `json:"nextRun"`
	LastRun        *time.Time `json:"lastRun"`
	LastRunStatus  *string    `json:"lastRunStatus"`
}

// BackupProfile is a flattened view of ent.BackupProfile with edges as direct properties.
type BackupProfile struct {
	ID                       int                           `json:"id"`
	CreatedAt                time.Time                     `json:"createdAt"`
	UpdatedAt                time.Time                     `json:"updatedAt"`
	Name                     string                        `json:"name"`
	Prefix                   string                        `json:"prefix"`
	BackupPaths              []string                      `json:"backupPaths"`
	ExcludePaths             []string                      `json:"excludePaths"`
	ExcludeCaches            bool                          `json:"excludeCaches"`
	Icon                     backupprofile.Icon            `json:"icon"`
	CompressionMode          backupprofile.CompressionMode `json:"compressionMode"`
	CompressionLevel         *int                          `json:"compressionLevel"`
	DataSectionCollapsed     bool                          `json:"dataSectionCollapsed"`
	ScheduleSectionCollapsed bool                          `json:"scheduleSectionCollapsed"`
	AdvancedSectionCollapsed bool                          `json:"advancedSectionCollapsed"`

	// Flattened edges (direct properties instead of .Edges.X)
	Repositories   []RepositorySummary `json:"repositories"`
	BackupSchedule *BackupSchedule     `json:"backupSchedule"`
	PruningRule    *PruningRule        `json:"pruningRule"`

	// Computed fields
	ArchiveCount int                `json:"archiveCount"`
	LastBackup   *types.LastBackup  `json:"lastBackup,omitempty"`
	LastAttempt  *types.LastAttempt `json:"lastAttempt,omitempty"`
}

// toBackupSchedule converts an ent.BackupSchedule to the custom BackupSchedule type
func toBackupSchedule(es *ent.BackupSchedule) *BackupSchedule {
	if es == nil {
		return nil
	}
	return &BackupSchedule{
		ID:              es.ID,
		CreatedAt:       es.CreatedAt,
		UpdatedAt:       es.UpdatedAt,
		Mode:            es.Mode,
		IntervalMinutes: es.IntervalMinutes,
		DailyAt:         es.DailyAt,
		Weekday:         es.Weekday,
		WeeklyAt:        es.WeeklyAt,
		Monthday:        es.Monthday,
		MonthlyAt:       es.MonthlyAt,
		NextRun:         es.NextRun,
		LastRun:         es.LastRun,
		LastRunStatus:   es.LastRunStatus,
	}
}

// toPruningRule converts an ent.PruningRule to the custom PruningRule type
func toPruningRule(ep *ent.PruningRule) *PruningRule {
	if ep == nil {
		return nil
	}
	return &PruningRule{
		ID:             ep.ID,
		CreatedAt:      ep.CreatedAt,
		UpdatedAt:      ep.UpdatedAt,
		IsEnabled:      ep.IsEnabled,
		KeepHourly:     ep.KeepHourly,
		KeepDaily:      ep.KeepDaily,
		KeepWeekly:     ep.KeepWeekly,
		KeepMonthly:    ep.KeepMonthly,
		KeepYearly:     ep.KeepYearly,
		KeepWithinDays: ep.KeepWithinDays,
		NextRun:        ep.NextRun,
		LastRun:        ep.LastRun,
		LastRunStatus:  ep.LastRunStatus,
	}
}

// toBackupProfile converts an ent.BackupProfile to the custom BackupProfile type
func (s *Service) toBackupProfile(ctx context.Context, ep *ent.BackupProfile) (*BackupProfile, error) {
	// Convert repositories to summaries
	repos := make([]RepositorySummary, 0)
	if ep.Edges.Repositories != nil {
		for _, r := range ep.Edges.Repositories {
			if r != nil {
				repos = append(repos, RepositorySummary{ID: r.ID, Name: r.Name})
			}
		}
	}

	// Get archive count
	archiveCount, err := s.db.Archive.Query().
		Where(archive.HasBackupProfileWith(backupprofile.ID(ep.ID))).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count archives: %w", err)
	}

	return &BackupProfile{
		ID:                       ep.ID,
		CreatedAt:                ep.CreatedAt,
		UpdatedAt:                ep.UpdatedAt,
		Name:                     ep.Name,
		Prefix:                   ep.Prefix,
		BackupPaths:              ep.BackupPaths,
		ExcludePaths:             ep.ExcludePaths,
		ExcludeCaches:            ep.ExcludeCaches,
		Icon:                     ep.Icon,
		CompressionMode:          ep.CompressionMode,
		CompressionLevel:         ep.CompressionLevel,
		DataSectionCollapsed:     ep.DataSectionCollapsed,
		ScheduleSectionCollapsed: ep.ScheduleSectionCollapsed,
		AdvancedSectionCollapsed: ep.AdvancedSectionCollapsed,
		Repositories:             repos,
		BackupSchedule:           toBackupSchedule(ep.Edges.BackupSchedule),
		PruningRule:              toPruningRule(ep.Edges.PruningRule),
		ArchiveCount:             archiveCount,
		LastBackup:               s.getLastBackup(ctx, ep.ID),
		LastAttempt:              s.getLastAttempt(ctx, ep.ID),
	}, nil
}

// getLastErrorNotification returns the most recent error notification for a backup profile
func (s *Service) getLastErrorNotification(ctx context.Context, backupProfileID int) *ent.Notification {
	notificationEnt, err := s.db.Notification.Query().
		Where(
			notification.HasBackupProfileWith(backupprofile.ID(backupProfileID)),
			notification.TypeIn(
				notification.TypeFailedBackupRun,
				notification.TypeFailedPruningRun,
			),
		).
		Order(ent.Desc(notification.FieldCreatedAt)).
		First(ctx)

	if err != nil {
		return nil
	}
	return notificationEnt
}

// getLastBackup returns info about the last successful backup for a backup profile
func (s *Service) getLastBackup(ctx context.Context, backupProfileID int) *types.LastBackup {
	lastArchive, err := s.db.Archive.Query().
		Where(archive.HasBackupProfileWith(backupprofile.ID(backupProfileID))).
		Order(ent.Desc(archive.FieldCreatedAt)).
		First(ctx)

	if err != nil {
		return nil // No successful backups yet
	}

	result := &types.LastBackup{
		Timestamp: &lastArchive.CreatedAt,
	}
	if lastArchive.WarningMessage != nil {
		result.WarningMessage = *lastArchive.WarningMessage
	}
	return result
}

// getLastAttempt returns info about the most recent attempt for a backup profile
func (s *Service) getLastAttempt(ctx context.Context, backupProfileID int) *types.LastAttempt {
	// Get latest archive
	lastArchive, _ := s.db.Archive.Query().
		Where(archive.HasBackupProfileWith(backupprofile.ID(backupProfileID))).
		Order(ent.Desc(archive.FieldCreatedAt)).
		First(ctx)

	// Get latest error notification
	errorNotification := s.getLastErrorNotification(ctx, backupProfileID)

	// Determine which is more recent
	if errorNotification != nil {
		// Check if error is newer than archive (or no archive exists)
		if lastArchive == nil || errorNotification.CreatedAt.After(lastArchive.CreatedAt) {
			return &types.LastAttempt{
				Status:    types.BackupStatusError,
				Timestamp: &errorNotification.CreatedAt,
				Message:   errorNotification.Message,
			}
		}
	}

	if lastArchive == nil {
		return nil // No attempts yet
	}

	// Archive is the most recent attempt
	if lastArchive.WarningMessage != nil {
		return &types.LastAttempt{
			Status:    types.BackupStatusWarning,
			Timestamp: &lastArchive.CreatedAt,
			Message:   *lastArchive.WarningMessage,
		}
	}

	return &types.LastAttempt{
		Status:    types.BackupStatusSuccess,
		Timestamp: &lastArchive.CreatedAt,
	}
}

// toBackupProfiles converts a slice of ent.BackupProfile to custom BackupProfile types
func (s *Service) toBackupProfiles(ctx context.Context, entProfiles []*ent.BackupProfile) ([]*BackupProfile, error) {
	profiles := make([]*BackupProfile, 0, len(entProfiles))
	for _, ep := range entProfiles {
		p, err := s.toBackupProfile(ctx, ep)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// ToEnt converts a BackupSchedule to ent.BackupSchedule for save operations
func (s *BackupSchedule) ToEnt() *ent.BackupSchedule {
	if s == nil {
		return nil
	}
	return &ent.BackupSchedule{
		ID:              s.ID,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
		Mode:            s.Mode,
		IntervalMinutes: s.IntervalMinutes,
		DailyAt:         s.DailyAt,
		Weekday:         s.Weekday,
		WeeklyAt:        s.WeeklyAt,
		Monthday:        s.Monthday,
		MonthlyAt:       s.MonthlyAt,
		NextRun:         s.NextRun,
		LastRun:         s.LastRun,
		LastRunStatus:   s.LastRunStatus,
	}
}

// ToEnt converts a PruningRule to ent.PruningRule for save operations
func (p *PruningRule) ToEnt() *ent.PruningRule {
	if p == nil {
		return nil
	}
	return &ent.PruningRule{
		ID:             p.ID,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
		IsEnabled:      p.IsEnabled,
		KeepHourly:     p.KeepHourly,
		KeepDaily:      p.KeepDaily,
		KeepWeekly:     p.KeepWeekly,
		KeepMonthly:    p.KeepMonthly,
		KeepYearly:     p.KeepYearly,
		KeepWithinDays: p.KeepWithinDays,
		NextRun:        p.NextRun,
		LastRun:        p.LastRun,
		LastRunStatus:  p.LastRunStatus,
	}
}

/***********************************/
/********** Compression ************/
/***********************************/

// validateCompression validates compression mode and level combinations
func validateCompression(mode backupprofile.CompressionMode, level *int) error {
	// Modes that don't support levels
	noLevelModes := map[backupprofile.CompressionMode]bool{
		backupprofile.CompressionModeNone: true,
		backupprofile.CompressionModeLz4:  true,
		backupprofile.CompressionModeZstd: false,
		backupprofile.CompressionModeZlib: false,
		backupprofile.CompressionModeLzma: false,
	}

	if noLevelModes[mode] && level != nil {
		return fmt.Errorf("compression mode '%s' does not support compression level", mode)
	}

	// Modes that require a level (database constraint)
	requiresLevel := map[backupprofile.CompressionMode]bool{
		backupprofile.CompressionModeNone: false,
		backupprofile.CompressionModeLz4:  false,
		backupprofile.CompressionModeZstd: true,
		backupprofile.CompressionModeZlib: true,
		backupprofile.CompressionModeLzma: true,
	}

	if requiresLevel[mode] && level == nil {
		return fmt.Errorf("compression mode '%s' requires a compression level", mode)
	}

	// Validate level ranges for modes that support them
	if level != nil {
		switch mode {
		case backupprofile.CompressionModeZstd:
			if *level < 1 || *level > 22 {
				return fmt.Errorf("zstd compression level must be between 1 and 22, got %d", *level)
			}
		case backupprofile.CompressionModeZlib:
			if *level < 0 || *level > 9 {
				return fmt.Errorf("zlib compression level must be between 0 and 9, got %d", *level)
			}
		case backupprofile.CompressionModeLzma:
			if *level < 0 || *level > 6 {
				return fmt.Errorf("lzma compression level must be between 0 and 6, got %d", *level)
			}
		case backupprofile.CompressionModeNone, backupprofile.CompressionModeLz4:
			// These modes don't support levels, already validated above
		}
	}

	return nil
}

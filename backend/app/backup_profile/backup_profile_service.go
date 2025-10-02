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
	"github.com/loomi-labs/arco/backend/ent/repository"
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

func (s *Service) GetBackupProfile(ctx context.Context, id int) (*ent.BackupProfile, error) {
	s.mustHaveDB()
	return s.db.BackupProfile.
		Query().
		WithRepositories().
		WithBackupSchedule().
		WithPruningRule().
		Where(backupprofile.ID(id)).
		Only(ctx)
}

func (s *Service) GetBackupProfiles(ctx context.Context) ([]*ent.BackupProfile, error) {
	s.mustHaveDB()
	return s.db.BackupProfile.
		Query().
		WithRepositories().
		WithBackupSchedule().
		WithPruningRule().
		Order(func(sel *sql.Selector) {
			// Order by name, case-insensitive
			sel.OrderExpr(sql.Expr(fmt.Sprintf("%s COLLATE NOCASE", backupprofile.FieldName)))
		}).
		All(ctx)
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

func (s *Service) NewBackupProfile(ctx context.Context) (*ent.BackupProfile, error) {
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
	schedule := &ent.BackupSchedule{
		Mode:      backupschedule.ModeHourly,
		DailyAt:   firstDayOfMonthAtNine,
		Weekday:   backupschedule.WeekdayMonday,
		WeeklyAt:  firstDayOfMonthAtNine,
		Monthday:  1,
		MonthlyAt: firstDayOfMonthAtNine,
	}

	pruningRule := &ent.PruningRule{
		IsEnabled:      false,
		KeepHourly:     defaultPruningOption.KeepHourly,
		KeepDaily:      defaultPruningOption.KeepDaily,
		KeepWeekly:     defaultPruningOption.KeepWeekly,
		KeepMonthly:    defaultPruningOption.KeepMonthly,
		KeepYearly:     defaultPruningOption.KeepYearly,
		KeepWithinDays: 30,
	}

	return &ent.BackupProfile{
		ID:           0,
		Name:         "",
		Prefix:       "",
		BackupPaths:  make([]string, 0),
		ExcludePaths: make([]string, 0),
		Icon:         selectedIcon,
		Edges: ent.BackupProfileEdges{
			BackupSchedule: schedule,
			PruningRule:    pruningRule,
		},
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

func (s *Service) CreateBackupProfile(ctx context.Context, backup ent.BackupProfile, repositoryIds []int) (*ent.BackupProfile, error) {
	s.mustHaveDB()
	s.log.Debug(fmt.Sprintf("Creating backup profile %d", backup.ID))
	return s.db.BackupProfile.
		Create().
		SetName(backup.Name).
		SetPrefix(backup.Prefix).
		SetBackupPaths(backup.BackupPaths).
		SetExcludePaths(backup.ExcludePaths).
		SetIcon(backup.Icon).
		AddRepositoryIDs(repositoryIds...).
		Save(ctx)
}

func (s *Service) UpdateBackupProfile(ctx context.Context, backup ent.BackupProfile) (*ent.BackupProfile, error) {
	s.mustHaveDB()
	s.log.Debug(fmt.Sprintf("Updating backup profile %d", backup.ID))
	return s.db.BackupProfile.
		UpdateOneID(backup.ID).
		SetName(backup.Name).
		SetIcon(backup.Icon).
		SetBackupPaths(backup.BackupPaths).
		SetExcludePaths(backup.ExcludePaths).
		SetDataSectionCollapsed(backup.DataSectionCollapsed).
		SetScheduleSectionCollapsed(backup.ScheduleSectionCollapsed).
		Save(ctx)
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
		for _, repo := range backupProfile.Edges.Repositories {
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
	assert.NotNil(bs.Edges.Repositories, "backup profile does not have repositories")
	for _, r := range bs.Edges.Repositories {
		if r.ID == repositoryId {
			return fmt.Errorf("repository is already in the backup profile")
		}
	}
	return s.db.BackupProfile.
		UpdateOneID(backupProfileId).
		AddRepositoryIDs(repositoryId).
		Exec(ctx)
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

	return s.db.BackupProfile.
		UpdateOneID(backupProfileId).
		RemoveRepositoryIDs(repositoryId).
		Exec(ctx)
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

func (s *Service) SaveBackupSchedule(ctx context.Context, backupProfileId int, schedule ent.BackupSchedule) error {
	s.mustHaveDB()
	s.log.Debug(fmt.Sprintf("Saving backup schedule for backup profile %d", backupProfileId))

	defer s.sendBackupScheduleChanged()
	doesExist, err := s.db.BackupSchedule.
		Query().
		Where(backupschedule.HasBackupProfileWith(backupprofile.ID(backupProfileId))).
		Exist(ctx)
	if err != nil {
		return err
	}

	var nextRun *time.Time
	if schedule.Mode != backupschedule.ModeDisabled {
		nr, err := getNextBackupTime(&schedule, time.Now())
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
			SetMode(schedule.Mode).
			SetDailyAt(schedule.DailyAt).
			SetWeeklyAt(schedule.WeeklyAt).
			SetWeekday(schedule.Weekday).
			SetMonthlyAt(schedule.MonthlyAt).
			SetMonthday(schedule.Monthday).
			ClearNextRun().
			SetNillableNextRun(nextRun).
			Exec(ctx)
	}
	return s.db.BackupSchedule.
		Create().
		SetMode(schedule.Mode).
		SetDailyAt(schedule.DailyAt).
		SetWeeklyAt(schedule.WeeklyAt).
		SetWeekday(schedule.Weekday).
		SetMonthlyAt(schedule.MonthlyAt).
		SetMonthday(schedule.Monthday).
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

func (s *Service) SavePruningRule(ctx context.Context, backupId int, rule ent.PruningRule) (*ent.PruningRule, error) {
	s.mustHaveDB()
	defer s.sendPruningRuleChanged()

	backupProfile, err := s.GetBackupProfile(ctx, backupId)
	if err != nil {
		return nil, err
	}

	nextRun := getNextPruneTime(backupProfile.Edges.BackupSchedule, time.Now())

	if backupProfile.Edges.PruningRule != nil {
		s.log.Debug(fmt.Sprintf("Updating pruning rule %d for backup profile %d", rule.ID, backupId))
		return s.db.PruningRule.
			// We ignore the ID from the given rule and get it from the db directly
			UpdateOneID(backupProfile.Edges.PruningRule.ID).
			SetIsEnabled(rule.IsEnabled).
			SetKeepHourly(rule.KeepHourly).
			SetKeepDaily(rule.KeepDaily).
			SetKeepWeekly(rule.KeepWeekly).
			SetKeepMonthly(rule.KeepMonthly).
			SetKeepYearly(rule.KeepYearly).
			SetKeepWithinDays(rule.KeepWithinDays).
			SetNextRun(nextRun).
			Save(ctx)
	}
	s.log.Debug(fmt.Sprintf("Creating pruning rule for backup profile %d", backupId))
	return s.db.PruningRule.
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

func (s *Service) sendPruningRuleChanged() {
	s.log.Debug("Sending pruning rule changed event")
	if s.pruningScheduleChangedCh == nil {
		return
	}
	s.pruningScheduleChangedCh <- struct{}{}
}

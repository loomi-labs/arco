package backup_profile

import (
	"context"
	"entgo.io/ent/dialect/sql"
	"errors"
	"fmt"
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
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"
)

// Service contains the business logic for backup profiles
type Service struct {
	log                      *zap.SugaredLogger
	db                       *ent.Client
	state                    *state.State
	config                   *types.Config
	eventEmitter             types.EventEmitter
	ctx                      context.Context
	backupScheduleChangedCh  chan struct{}
	pruningScheduleChangedCh chan struct{}
}

// ServiceInternal provides backend-only methods that should not be exposed to frontend
type ServiceInternal struct {
	*Service
}

// NewService creates a new backup profile service
func NewService(ctx context.Context, log *zap.SugaredLogger, state *state.State, config *types.Config) *ServiceInternal {
	return &ServiceInternal{
		Service: &Service{
			ctx:    ctx,
			log:    log,
			state:  state,
			config: config,
		},
	}
}

// Init initializes the service with remaining dependencies
func (si *ServiceInternal) Init(db *ent.Client, eventEmitter types.EventEmitter) {
	si.db = db
	si.eventEmitter = eventEmitter
}

// SetBackupScheduleChangedCh sets the channel for backup schedule changes
func (si *ServiceInternal) SetBackupScheduleChangedCh(ch chan struct{}) {
	si.backupScheduleChangedCh = ch
}

// SetPruningScheduleChangedCh sets the channel for pruning schedule changes  
func (si *ServiceInternal) SetPruningScheduleChangedCh(ch chan struct{}) {
	si.pruningScheduleChangedCh = ch
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

func (s *Service) GetBackupProfile(id int) (*ent.BackupProfile, error) {
	s.mustHaveDB()
	return s.db.BackupProfile.
		Query().
		WithRepositories().
		WithBackupSchedule().
		WithPruningRule().
		Where(backupprofile.ID(id)).
		Only(s.ctx)
}

func (s *Service) GetBackupProfiles() ([]*ent.BackupProfile, error) {
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
		All(s.ctx)
}

type BackupProfileFilter struct {
	Id              int    `json:"id,omitempty"`
	Name            string `json:"name"`
	IsAllFilter     bool   `json:"isAllFilter"`
	IsUnknownFilter bool   `json:"isUnknownFilter"`
}

func (s *Service) GetBackupProfileFilterOptions(repoId int) ([]BackupProfileFilter, error) {
	s.mustHaveDB()
	profiles, err := s.db.BackupProfile.
		Query().
		Where(backupprofile.HasRepositoriesWith(repository.ID(repoId))).
		Order(ent.Desc(backupprofile.FieldName)).
		All(s.ctx)
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
		Exist(s.ctx)
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

func (s *Service) NewBackupProfile() (*ent.BackupProfile, error) {
	s.mustHaveDB()
	// Choose the first icon that is not already in use
	all, err := s.db.BackupProfile.
		Query().
		Select(backupprofile.FieldIcon).
		All(s.ctx)
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

func (s *Service) GetPrefixSuggestion(name string) (string, error) {
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
		Exist(s.ctx)
	if err != nil {
		return "", err
	}
	if exist {
		// If the prefix already exists, we create a new one by appending a random number
		prefix = fmt.Sprintf("%s%04d", prefix, rand.Intn(1000))
		return s.GetPrefixSuggestion(prefix)
	}
	return fullPrefix, nil
}

func (s *Service) CreateBackupProfile(backup ent.BackupProfile, repositoryIds []int) (*ent.BackupProfile, error) {
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
		Save(s.ctx)
}

func (s *Service) UpdateBackupProfile(backup ent.BackupProfile) (*ent.BackupProfile, error) {
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
		Save(s.ctx)
}

// DeleteBackupProfile deletes a backup profile and optionally its archives
// The deleteJobs parameter contains functions to execute Borg delete operations
func (s *Service) DeleteBackupProfile(backupProfileId int, deleteJobs []func()) error {
	s.mustHaveDB()
	err := s.db.BackupProfile.
		DeleteOneID(backupProfileId).
		Exec(s.ctx)
	if err != nil {
		return err
	}
	s.eventEmitter.EmitEvent(s.ctx, types.EventBackupProfileDeleted.String())

	// Execute the delete jobs
	for _, fn := range deleteJobs {
		fn()
	}

	return nil
}

func (s *Service) AddRepositoryToBackupProfile(backupProfileId int, repositoryId int) error {
	s.mustHaveDB()
	bs, err := s.GetBackupProfile(backupProfileId)
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
		Exec(s.ctx)
}

// RemoveRepositoryFromBackupProfile removes a repository from a backup profile
// The deleteJob parameter contains a function to execute Borg delete operations if needed
func (s *Service) RemoveRepositoryFromBackupProfile(backupProfileId int, repositoryId int, deleteJob func()) error {
	s.mustHaveDB()
	s.log.Debug(fmt.Sprintf("Removing repository %d from backup profile %d", repositoryId, backupProfileId))

	// Check if the repository is the only one in the backup profile
	cnt, err := s.db.Repository.
		Query().
		Where(repository.HasBackupProfilesWith(backupprofile.ID(backupProfileId))).
		Count(s.ctx)
	if err != nil {
		return err
	}
	if cnt <= 1 {
		assert.NotEqual(cnt, 0, "backup profile does not have repositories")
		return fmt.Errorf("cannot remove the only repository from the backup profile")
	}

	// Execute the delete job if provided
	if deleteJob != nil {
		go deleteJob()
	}

	return s.db.BackupProfile.
		UpdateOneID(backupProfileId).
		RemoveRepositoryIDs(repositoryId).
		Exec(s.ctx)
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

func (s *Service) SaveBackupSchedule(backupProfileId int, schedule ent.BackupSchedule) error {
	s.mustHaveDB()
	s.log.Debug(fmt.Sprintf("Saving backup schedule for backup profile %d", backupProfileId))

	defer s.sendBackupScheduleChanged()
	doesExist, err := s.db.BackupSchedule.
		Query().
		Where(backupschedule.HasBackupProfileWith(backupprofile.ID(backupProfileId))).
		Exist(s.ctx)
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
			Exec(s.ctx)
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
		Exec(s.ctx)
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

func (s *Service) SavePruningRule(backupId int, rule ent.PruningRule) (*ent.PruningRule, error) {
	s.mustHaveDB()
	defer s.sendPruningRuleChanged()

	backupProfile, err := s.GetBackupProfile(backupId)
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
			Save(s.ctx)
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
		Save(s.ctx)
}

func (s *Service) sendPruningRuleChanged() {
	s.log.Debug("Sending pruning rule changed event")
	if s.pruningScheduleChangedCh == nil {
		return
	}
	s.pruningScheduleChangedCh <- struct{}{}
}

// Helper functions (these need to be imported from schedule_utils.go or defined here)
func getNextBackupTime(schedule *ent.BackupSchedule, now time.Time) (time.Time, error) {
	// TODO: Implement this function or import it from schedule_utils
	// This is a placeholder implementation
	switch schedule.Mode {
	case backupschedule.ModeHourly:
		return now.Add(time.Hour), nil
	case backupschedule.ModeDaily:
		return schedule.DailyAt, nil
	case backupschedule.ModeWeekly:
		return schedule.WeeklyAt, nil
	case backupschedule.ModeMonthly:
		return schedule.MonthlyAt, nil
	default:
		return time.Time{}, errors.New("invalid schedule mode")
	}
}

func getNextPruneTime(schedule *ent.BackupSchedule, now time.Time) time.Time {
	// TODO: Implement this function or import it from schedule_utils
	// This is a placeholder implementation
	return now.Add(24 * time.Hour)
}
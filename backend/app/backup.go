package app

import (
	"entgo.io/ent/dialect/sql"
	"errors"
	"fmt"
	"github.com/eminarican/safetypes"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	borgtypes "github.com/loomi-labs/arco/backend/borg/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/archive"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/backupschedule"
	"github.com/loomi-labs/arco/backend/ent/notification"
	"github.com/loomi-labs/arco/backend/ent/pruningrule"
	"github.com/loomi-labs/arco/backend/ent/repository"
	"github.com/loomi-labs/arco/backend/util"
	"github.com/negrel/assert"
	"github.com/wailsapp/wails/v3/pkg/application"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"
)

/***********************************/
/********** Backup Profile *********/
/***********************************/

func (b *BackupClient) GetBackupProfile(id int) (*ent.BackupProfile, error) {
	return b.db.BackupProfile.
		Query().
		WithRepositories().
		WithBackupSchedule().
		WithPruningRule().
		Where(backupprofile.ID(id)).
		Only(b.ctx)
}

func (b *BackupClient) GetBackupProfiles() ([]*ent.BackupProfile, error) {
	return b.db.BackupProfile.
		Query().
		WithRepositories().
		WithBackupSchedule().
		WithPruningRule().
		Order(func(s *sql.Selector) {
			// Order by name, case-insensitive
			s.OrderExpr(sql.Expr(fmt.Sprintf("%s COLLATE NOCASE", backupprofile.FieldName)))
		}).
		All(b.ctx)
}

type BackupProfileFilter struct {
	Id              int    `json:"id,omitempty"`
	Name            string `json:"name"`
	IsAllFilter     bool   `json:"isAllFilter"`
	IsUnknownFilter bool   `json:"isUnknownFilter"`
}

func (b *BackupClient) GetBackupProfileFilterOptions(repoId int) ([]BackupProfileFilter, error) {
	profiles, err := b.db.BackupProfile.
		Query().
		Where(backupprofile.HasRepositoriesWith(repository.ID(repoId))).
		Order(ent.Desc(backupprofile.FieldName)).
		All(b.ctx)
	if err != nil {
		return nil, err
	}

	filters := make([]BackupProfileFilter, len(profiles))
	for i, p := range profiles {
		filters[i] = BackupProfileFilter{Id: p.ID, Name: p.Name}
	}

	hasForeignArchives, err := b.db.Repository.
		Query().
		Where(repository.And(
			repository.ID(repoId),
			repository.HasArchivesWith(archive.Not(archive.HasBackupProfile())),
		)).
		Exist(b.ctx)
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

func (b *BackupClient) NewBackupProfile() (*ent.BackupProfile, error) {
	// Choose the first icon that is not already in use
	all, err := b.db.BackupProfile.
		Query().
		Select(backupprofile.FieldIcon).
		All(b.ctx)
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

func (b *BackupClient) GetDirectorySuggestions() []string {
	home, _ := os.UserHomeDir()
	if home != "" {
		return []string{home}
	}
	return []string{}
}

func (b *BackupClient) DoesPathExist(path string) bool {
	_, err := os.Stat(util.ExpandPath(path))
	return err == nil
}

func (b *BackupClient) IsDirectory(path string) bool {
	info, err := os.Stat(util.ExpandPath(path))
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (b *BackupClient) IsDirectoryEmpty(path string) bool {
	path = util.ExpandPath(path)
	if !b.IsDirectory(path) {
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

func (b *BackupClient) CreateDirectory(path string) error {
	b.log.Debug(fmt.Sprintf("Creating directory %s", path))
	return os.MkdirAll(util.ExpandPath(path), 0755)
}

func (b *BackupClient) GetPrefixSuggestion(name string) (string, error) {
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

	exist, err := b.db.BackupProfile.
		Query().
		Where(backupprofile.Prefix(fullPrefix)).
		Exist(b.ctx)
	if err != nil {
		return "", err
	}
	if exist {
		// If the prefix already exists, we create a new one by appending a random number
		prefix = fmt.Sprintf("%s%04d", prefix, rand.Intn(1000))
		return b.GetPrefixSuggestion(prefix)
	}
	return fullPrefix, nil
}

func (b *BackupClient) CreateBackupProfile(backup ent.BackupProfile, repositoryIds []int) (*ent.BackupProfile, error) {
	b.log.Debug(fmt.Sprintf("Creating backup profile %d", backup.ID))
	return b.db.BackupProfile.
		Create().
		SetName(backup.Name).
		SetPrefix(backup.Prefix).
		SetBackupPaths(backup.BackupPaths).
		SetExcludePaths(backup.ExcludePaths).
		SetIcon(backup.Icon).
		AddRepositoryIDs(repositoryIds...).
		Save(b.ctx)
}

func (b *BackupClient) UpdateBackupProfile(backup ent.BackupProfile) (*ent.BackupProfile, error) {
	b.log.Debug(fmt.Sprintf("Updating backup profile %d", backup.ID))
	return b.db.BackupProfile.
		UpdateOneID(backup.ID).
		SetName(backup.Name).
		SetIcon(backup.Icon).
		SetBackupPaths(backup.BackupPaths).
		SetExcludePaths(backup.ExcludePaths).
		SetDataSectionCollapsed(backup.DataSectionCollapsed).
		SetScheduleSectionCollapsed(backup.ScheduleSectionCollapsed).
		Save(b.ctx)
}

func (b *BackupClient) DeleteBackupProfile(backupProfileId int, deleteArchives bool) error {
	backupProfile, err := b.GetBackupProfile(backupProfileId)
	if err != nil {
		return err
	}

	var deleteJobs []func()
	// If deleteArchives is true, we prepare a delete job for each repository
	if deleteArchives {
		for _, r := range backupProfile.Edges.Repositories {
			repo := r // Capture loop variable
			bId := types.BackupId{
				BackupProfileId: backupProfileId,
				RepositoryId:    repo.ID,
			}
			deleteJobs = append(deleteJobs, func() {
				go func() {
					_, err := b.runBorgDelete(bId, repo.URL, repo.Password, backupProfile.Prefix)
					if err != nil {
						b.log.Error(fmt.Sprintf("Delete job failed: %s", err))
					}
				}()
			})
		}
	}

	err = b.db.BackupProfile.
		DeleteOneID(backupProfileId).
		Exec(b.ctx)
	if err != nil {
		return err
	}
	b.eventEmitter.EmitEvent(b.ctx, types.EventBackupProfileDeleted.String())

	// Execute the delete jobs
	for _, fn := range deleteJobs {
		fn()
	}

	return nil
}

func (b *BackupClient) AddRepositoryToBackupProfile(backupProfileId int, repositoryId int) error {
	bs, err := b.GetBackupProfile(backupProfileId)
	if err != nil {
		return err
	}
	assert.NotNil(bs.Edges.Repositories, "backup profile does not have repositories")
	for _, r := range bs.Edges.Repositories {
		if r.ID == repositoryId {
			return fmt.Errorf("repository is already in the backup profile")
		}
	}
	return b.db.BackupProfile.
		UpdateOneID(backupProfileId).
		AddRepositoryIDs(repositoryId).
		Exec(b.ctx)
}

func (b *BackupClient) RemoveRepositoryFromBackupProfile(backupProfileId int, repositoryId int, deleteArchives bool) error {
	b.log.Debug(fmt.Sprintf("Removing repository %d from backup profile %d (deleteArchives: %t)", repositoryId, backupProfileId, deleteArchives))

	// Check if the repository is the only one in the backup profile
	cnt, err := b.db.Repository.
		Query().
		Where(repository.HasBackupProfilesWith(backupprofile.ID(backupProfileId))).
		Count(b.ctx)
	if err != nil {
		return err
	}
	if cnt <= 1 {
		assert.NotEqual(cnt, 0, "backup profile does not have repositories")
		return fmt.Errorf("cannot remove the only repository from the backup profile")
	}

	// Get the backup profile with the repository
	backupProfile, err := b.db.BackupProfile.
		Query().
		Where(backupprofile.And(
			backupprofile.ID(backupProfileId),
			backupprofile.HasRepositoriesWith(repository.ID(repositoryId)),
		)).
		WithRepositories(func(q *ent.RepositoryQuery) {
			q.Where(repository.ID(repositoryId))
		}).
		Only(b.ctx)
	if err != nil {
		return err
	}
	assert.NotEmpty(backupProfile.Edges.Repositories, "repository does not have the backup profile")

	// If deleteArchives is true, we run a delete job for the repository
	if deleteArchives {
		bId := types.BackupId{
			BackupProfileId: backupProfileId,
			RepositoryId:    repositoryId,
		}
		repo := backupProfile.Edges.Repositories[0]
		if repo.ID == repositoryId {
			location, password, prefix := repo.URL, repo.Password, backupProfile.Prefix
			go func() {
				_, err := b.runBorgDelete(bId, location, password, prefix)
				if err != nil {
					b.log.Error(fmt.Sprintf("Delete job failed: %s", err))
				}
			}()
		}
	}

	return b.db.BackupProfile.
		UpdateOneID(backupProfileId).
		RemoveRepositoryIDs(repositoryId).
		Exec(b.ctx)
}

func (b *BackupClient) getRepoWithBackupProfile(repoId int, backupProfileId int) (*ent.Repository, error) {
	repo, err := b.db.Repository.
		Query().
		Where(repository.And(
			repository.ID(repoId),
			repository.HasBackupProfilesWith(backupprofile.ID(backupProfileId)),
		)).
		WithBackupProfiles(func(q *ent.BackupProfileQuery) {
			q.Limit(1)
			q.Where(backupprofile.ID(backupProfileId))
			q.WithPruningRule()
		}).
		Only(b.ctx)
	if err != nil {
		return nil, err
	}
	if len(repo.Edges.BackupProfiles) != 1 {
		return nil, fmt.Errorf("repository does not have the backup profile")
	}
	return repo, nil
}

/***********************************/
/********** Backup Functions *******/
/***********************************/

func (b *BackupClient) startBackupJob(bId types.BackupId) error {
	if canRun, reason := b.state.CanRunBackup(bId); !canRun {
		return errors.New(reason)
	}

	go func() {
		_, err := b.runBorgCreate(bId)
		if err != nil {
			b.log.Error(fmt.Sprintf("Backup job failed: %s", err))
		}
	}()

	return nil
}

func (b *BackupClient) StartBackupJobs(bIds []types.BackupId) error {
	var errs []error
	for _, bId := range bIds {
		err := b.startBackupJob(bId)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to start some backup jobs: %v", errs)
	}
	return nil
}

type SelectDirectoryData struct {
	Title      string `json:"title"`
	Message    string `json:"message"`
	ButtonText string `json:"buttonText"`
}

func (b *BackupClient) SelectDirectory(data SelectDirectoryData) (string, error) {
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

type BackupProgressResponse struct {
	BackupId types.BackupId           `json:"backupId"`
	Progress borgtypes.BackupProgress `json:"progress"`
	Found    bool                     `json:"found"`
}

func (b *BackupClient) abortBackupJob(id types.BackupId) error {
	b.state.SetBackupCancelled(b.ctx, id, true)
	return nil
}

func (b *BackupClient) AbortBackupJobs(bIds []types.BackupId) error {
	for _, bId := range bIds {
		if b.state.GetBackupState(bId).Status == state.BackupStatusRunning {
			err := b.abortBackupJob(bId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *BackupClient) GetState(bId types.BackupId) state.BackupState {
	return b.state.GetBackupState(bId)
}

func (b *BackupClient) GetBackupButtonStatus(bIds []types.BackupId) state.BackupButtonStatus {
	switch len(bIds) {
	case 0:
		return state.BackupButtonStatusRunBackup
	case 1:
		return b.state.GetBackupButtonStatus(bIds[0])
	default:
		return b.state.GetCombinedBackupButtonStatus(bIds)
	}
}

func (b *BackupClient) GetCombinedBackupProgress(bIds []types.BackupId) *borgtypes.BackupProgress {
	return b.state.GetCombinedBackupProgress(bIds)
}

func (b *BackupClient) GetLastBackupErrorMsg(bId types.BackupId) (string, error) {
	// Get the last notification for the backup profile and repository
	lastNotification, err := b.db.Notification.
		Query().
		Where(notification.And(
			notification.HasBackupProfileWith(backupprofile.ID(bId.BackupProfileId)),
			notification.HasRepositoryWith(repository.ID(bId.RepositoryId)),
		)).
		Order(ent.Desc(notification.FieldCreatedAt)).
		First(b.ctx)
	if err != nil && !ent.IsNotFound(err) {
		return "", err
	}
	if lastNotification != nil {
		// Check if there is a new archive since the last notification
		// If there is, we don't show the error message
		exist, err := b.db.Archive.
			Query().
			Where(archive.And(
				archive.HasBackupProfileWith(backupprofile.ID(bId.BackupProfileId)),
				archive.HasRepositoryWith(repository.ID(bId.RepositoryId)),
				archive.CreatedAtGT(lastNotification.CreatedAt),
			)).
			Exist(b.ctx)
		if err != nil && !ent.IsNotFound(err) {
			return "", err
		}
		if !exist {
			return lastNotification.Message, nil
		}
	}
	return "", nil
}

/***********************************/
/********** Backup Schedule ********/
/***********************************/

func (b *BackupClient) SaveBackupSchedule(backupProfileId int, schedule ent.BackupSchedule) error {
	b.log.Debug(fmt.Sprintf("Saving backup schedule for backup profile %d", backupProfileId))

	defer b.sendBackupScheduleChanged()
	doesExist, err := b.db.BackupSchedule.
		Query().
		Where(backupschedule.HasBackupProfileWith(backupprofile.ID(backupProfileId))).
		Exist(b.ctx)
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
		b.log.Debugf("Next run in %s", nextRun.Sub(time.Now()))
	}

	if doesExist {
		return b.db.BackupSchedule.
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
			Exec(b.ctx)
	}
	return b.db.BackupSchedule.
		Create().
		SetMode(schedule.Mode).
		SetDailyAt(schedule.DailyAt).
		SetWeeklyAt(schedule.WeeklyAt).
		SetWeekday(schedule.Weekday).
		SetMonthlyAt(schedule.MonthlyAt).
		SetMonthday(schedule.Monthday).
		SetNillableNextRun(nextRun).
		SetBackupProfileID(backupProfileId).
		Exec(b.ctx)
}

func (b *BackupClient) sendBackupScheduleChanged() {
	if b.backupScheduleChangedCh == nil {
		return
	}
	b.backupScheduleChangedCh <- struct{}{}
}

/***********************************/
/********** Other Functions ********/
/***********************************/

func (b *BackupClient) refreshRepoInfo(repoId int, url, password string) error {
	info, status := b.borg.Info(b.ctx, url, password)
	if !status.IsCompletedWithSuccess() {
		if status.HasBeenCanceled {
			return fmt.Errorf("repository info retrieval was cancelled")
		}
		return fmt.Errorf("failed to get repository info: %s", status.GetError())
	}
	if status.HasWarning() {
		// TODO(log-warning): log warning to user
		b.log.Warnf("Repository info retrieval completed with warning: %s", status.GetWarning())
	}
	if info == nil {
		return fmt.Errorf("failed to get repository info: response is nil")
	}
	return b.db.Repository.
		UpdateOneID(repoId).
		SetStatsTotalSize(info.Cache.Stats.TotalSize).
		SetStatsTotalCsize(info.Cache.Stats.TotalCSize).
		SetStatsTotalChunks(info.Cache.Stats.TotalChunks).
		SetStatsTotalUniqueChunks(info.Cache.Stats.TotalUniqueChunks).
		SetStatsUniqueCsize(info.Cache.Stats.UniqueCSize).
		SetStatsUniqueSize(info.Cache.Stats.UniqueSize).
		Exec(b.ctx)
}

func (b *BackupClient) addNewArchive(bId types.BackupId, archivePath, password string) error {
	info, status := b.borg.Info(b.ctx, archivePath, password)
	if !status.IsCompletedWithSuccess() {
		if status.HasBeenCanceled {
			return fmt.Errorf("repository info retrieval was cancelled")
		}
		return fmt.Errorf("failed to get archive info: %s", status.GetError())
	}
	if status.HasWarning() {
		// TODO(log-warning): log warning to user
		b.log.Warnf("Repository info retrieval completed with warning: %s", status.GetWarning())
	}
	if info == nil {
		return fmt.Errorf("failed to get archive info: response is nil")
	}
	if len(info.Archives) == 0 {
		return fmt.Errorf("no archives found")
	}
	createdAt := time.Time(info.Archives[0].Start)
	duration := time.Time(info.Archives[0].End).Sub(createdAt)
	_, err := b.db.Archive.
		Create().
		SetRepositoryID(bId.RepositoryId).
		SetBackupProfileID(bId.BackupProfileId).
		SetBorgID(info.Archives[0].ID).
		SetName(info.Archives[0].Name).
		SetCreatedAt(createdAt).
		SetDuration(duration.Seconds()).
		Save(b.ctx)
	return err
}

func (b *BackupClient) saveDbNotification(backupId types.BackupId, message string, notificationType notification.Type, action safetypes.Option[notification.Action]) error {
	return b.db.Notification.
		Create().
		SetMessage(message).
		SetType(notificationType).
		SetBackupProfileID(backupId.BackupProfileId).
		SetRepositoryID(backupId.RepositoryId).
		SetNillableAction(action.Value).
		Exec(b.ctx)
}

/***********************************/
/********** Borg Commands **********/
/***********************************/

type BackupResult string

const (
	BackupResultSuccess   BackupResult = "success"
	BackupResultCancelled BackupResult = "cancelled"
	BackupResultError     BackupResult = "error"
)

func (b BackupResult) String() string {
	return string(b)
}

// runBorgCreate runs the actual backup job.
// It is long running and should be run in a goroutine.
func (b *BackupClient) runBorgCreate(bId types.BackupId) (result BackupResult, err error) {
	repo, err := b.getRepoWithBackupProfile(bId.RepositoryId, bId.BackupProfileId)
	if err != nil {
		b.state.SetBackupError(b.ctx, bId, err, false, false)
		b.state.AddNotification(b.ctx, fmt.Sprintf("Failed to get repository: %s", err), types.LevelError)
		return BackupResultError, err
	}
	assert.NotEmpty(repo.Edges.BackupProfiles, "repository does not have backup profiles")
	backupProfile := repo.Edges.BackupProfiles[0]
	b.state.SetBackupWaiting(b.ctx, bId)

	repoLock := b.state.GetRepoLock(bId.RepositoryId)
	repoLock.Lock()         // We might wait here for other operations to finish
	defer repoLock.Unlock() // Unlock at the end

	// Wait to acquire the lock and then set the backup as running
	ctx := b.state.SetBackupRunning(b.ctx, bId)

	// Create go routine to receive progress info
	ch := make(chan borgtypes.BackupProgress)
	defer close(ch)
	go b.saveProgressInfo(bId, ch)

	archiveName, status := b.borg.Create(ctx, repo.URL, repo.Password, backupProfile.Prefix, backupProfile.BackupPaths, backupProfile.ExcludePaths, ch)
	if !status.IsCompletedWithSuccess() {
		if status.HasBeenCanceled {
			b.state.SetBackupCancelled(b.ctx, bId, true)
			return BackupResultCancelled, nil
		} else if status.HasError() && errors.Is(status.Error, borgtypes.ErrorLockTimeout) {
			retErr := fmt.Errorf("repository %s is locked", repo.Name)
			saveErr := b.saveDbNotification(bId, retErr.Error(), notification.TypeFailedBackupRun, safetypes.Some(notification.ActionUnlockRepository))
			if saveErr != nil {
				b.log.Error(fmt.Sprintf("Failed to save notification: %s", saveErr))
			}
			b.state.SetBackupError(b.ctx, bId, retErr, false, true)
			b.state.AddNotification(b.ctx, fmt.Sprintf("Backup job failed: repository %s is locked", repo.Name), types.LevelError)
			return BackupResultError, retErr
		} else {
			saveErr := b.saveDbNotification(bId, status.Error.Error(), notification.TypeFailedBackupRun, safetypes.None[notification.Action]())
			if saveErr != nil {
				b.log.Error(fmt.Sprintf("Failed to save notification: %s", saveErr))
			}
			b.state.SetBackupError(b.ctx, bId, status.Error, true, false)
			b.state.AddNotification(b.ctx, fmt.Sprintf("Backup job failed: %s", status.Error), types.LevelError)
			return BackupResultError, status.Error
		}
	} else {
		// Backup completed successfully
		defer b.state.SetBackupCompleted(b.ctx, bId, true)

		err = b.refreshRepoInfo(bId.RepositoryId, repo.URL, repo.Password)
		if err != nil {
			b.log.Error(fmt.Sprintf("Failed to get info for backup %d: %s", bId, err))
		}

		err = b.addNewArchive(bId, archiveName, repo.Password)
		if err != nil {
			b.log.Error(fmt.Sprintf("Failed to add new archive for backup %d: %s", bId, err))
		}

		pruningRule, pErr := b.db.PruningRule.
			Query().
			Where(pruningrule.And(
				pruningrule.HasBackupProfileWith(backupprofile.ID(bId.BackupProfileId)),
				pruningrule.IsEnabled(true),
			)).
			Only(b.ctx)
		if pErr != nil && !ent.IsNotFound(pErr) {
			b.log.Error(fmt.Sprintf("Failed to get pruning rule: %s", pErr))
		}
		if pruningRule != nil && pruningRule.IsEnabled {
			_, err = b.examinePrune(bId, safetypes.Some(pruningRule), true, true)
			if err != nil {
				b.log.Error(fmt.Sprintf("Failed to examine prune: %s", err))
			}
		}

		return BackupResultSuccess, nil
	}
}

type DeleteResult string

const (
	DeleteResultSuccess   DeleteResult = "success"
	DeleteResultCancelled DeleteResult = "cancelled"
	DeleteResultError     DeleteResult = "error"
)

func (b *BackupClient) runBorgDelete(bId types.BackupId, location, password, prefix string) (DeleteResult, error) {
	repoLock := b.state.GetRepoLock(bId.RepositoryId)
	repoLock.Lock()         // We might wait here for other operations to finish
	defer repoLock.Unlock() // Unlock at the end

	// Wait to acquire the lock and then set the repo as locked
	b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusDeleting)

	status := b.borg.DeleteArchives(b.ctx, location, password, prefix)
	if !status.IsCompletedWithSuccess() {
		if status.HasBeenCanceled {
			b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusIdle)
			return DeleteResultCancelled, nil
		} else if status.HasError() && errors.Is(status.Error, borgtypes.ErrorLockTimeout) {
			b.state.AddNotification(b.ctx, "Delete job failed: repository is locked", types.LevelError)
			b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusLocked)
			return DeleteResultError, status.Error
		} else {
			b.state.AddNotification(b.ctx, fmt.Sprintf("Delete job failed: %s", status.Error), types.LevelError)
			b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusIdle)
			return DeleteResultError, status.Error
		}
	} else {
		// Delete completed successfully
		defer b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusIdle)

		err := b.refreshRepoInfo(bId.RepositoryId, location, password)
		if err != nil {
			b.log.Error(fmt.Sprintf("Failed to get info for backup-profile %d: %s", bId, err))
		}

		_, err = b.repositoryService.RefreshArchivesWithoutLock(b.ctx, bId.RepositoryId)
		if err != nil {
			b.log.Error(fmt.Sprintf("Failed to refresh archives for backup-profile %d: %s", bId, err))
		}

		return DeleteResultSuccess, nil
	}
}

func (b *BackupClient) saveProgressInfo(id types.BackupId, ch chan borgtypes.BackupProgress) {
	for {
		select {
		case <-b.ctx.Done():
			return
		case progress, ok := <-ch:
			if !ok {
				// Channel is closed, break the loop
				return
			}
			b.state.UpdateBackupProgress(b.ctx, id, progress)
		}
	}
}

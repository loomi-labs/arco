package app

import (
	"entgo.io/ent/dialect/sql"
	"errors"
	"fmt"
	"github.com/eminarican/safetypes"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/borg"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/archive"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/backupschedule"
	"github.com/loomi-labs/arco/backend/ent/notification"
	"github.com/loomi-labs/arco/backend/ent/pruningrule"
	"github.com/loomi-labs/arco/backend/ent/repository"
	"github.com/loomi-labs/arco/backend/util"
	"github.com/negrel/assert"
	"github.com/wailsapp/wails/v2/pkg/runtime"
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

	schedule := &ent.BackupSchedule{
		Mode:      backupschedule.ModeHourly,
		DailyAt:   time.Date(0, 1, 1, 9, 0, 0, 0, time.Local),
		Weekday:   backupschedule.WeekdayMonday,
		WeeklyAt:  time.Date(0, 1, 1, 9, 0, 0, 0, time.Local),
		Monthday:  1,
		MonthlyAt: time.Date(0, 1, 1, 9, 0, 0, 0, time.Local),
	}

	pruningRule := &ent.PruningRule{
		IsEnabled:      false,
		KeepHourly:     12,
		KeepDaily:      7,
		KeepWeekly:     4,
		KeepMonthly:    6,
		KeepYearly:     1,
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
					_, err := b.runBorgDelete(bId, repo.Location, repo.Password, backupProfile.Prefix)
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
	backupProfile, err := b.GetBackupProfile(backupProfileId)
	if err != nil {
		return err
	}
	assert.NotNil(backupProfile.Edges.Repositories, "backup profile does not have repositories")
	if len(backupProfile.Edges.Repositories) == 1 {
		return fmt.Errorf("cannot remove the only repository from the backup profile")
	}
	if deleteArchives {
		bId := types.BackupId{
			BackupProfileId: backupProfileId,
			RepositoryId:    repositoryId,
		}
		for _, r := range backupProfile.Edges.Repositories {
			repo := r // Capture loop variable
			if repo.ID == repositoryId {
				location, password, prefix := repo.Location, repo.Password, backupProfile.Prefix
				go func() {
					_, err := b.runBorgDelete(bId, location, password, prefix)
					if err != nil {
						b.log.Error(fmt.Sprintf("Delete job failed: %s", err))
					}
				}()
			}
		}
	}

	err = b.db.BackupProfile.
		UpdateOneID(backupProfileId).
		RemoveRepositoryIDs(repositoryId).
		Exec(b.ctx)
	if err != nil {
		return err
	}
	return nil
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

func (b *BackupClient) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(b.ctx, runtime.OpenDialogOptions{})
}

type BackupProgressResponse struct {
	BackupId types.BackupId      `json:"backupId"`
	Progress borg.BackupProgress `json:"progress"`
	Found    bool                `json:"found"`
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

func (b *BackupClient) GetCombinedBackupProgress(bIds []types.BackupId) *borg.BackupProgress {
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
	info, err := b.borg.Info(url, password)
	if err != nil {
		return err
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

func (b *BackupClient) addNewArchive(bId types.BackupId, archiveName, password string) error {
	info, err := b.borg.Info(archiveName, password)
	if err != nil {
		return err
	}
	if len(info.Archives) == 0 {
		return fmt.Errorf("no archives found")
	}

	_, err = b.db.Archive.
		Create().
		SetRepositoryID(bId.RepositoryId).
		SetBackupProfileID(bId.BackupProfileId).
		SetBorgID(info.Archives[0].ID).
		SetName(info.Archives[0].Name).
		SetCreatedAt(time.Time(info.Archives[0].Start)).
		SetDuration(time.Time(info.Archives[0].Duration)).
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
	ch := make(chan borg.BackupProgress)
	defer close(ch)
	go b.saveProgressInfo(bId, ch)

	archiveName, err := b.borg.Create(ctx, repo.Location, repo.Password, backupProfile.Prefix, backupProfile.BackupPaths, backupProfile.ExcludePaths, ch)
	if err != nil {
		if errors.As(err, &borg.CancelErr{}) {
			b.state.SetBackupCancelled(b.ctx, bId, true)
			return BackupResultCancelled, nil
		} else if errors.As(err, &borg.LockTimeout{}) {
			err = fmt.Errorf("repository %s is locked", repo.Name)
			saveErr := b.saveDbNotification(bId, err.Error(), notification.TypeFailedBackupRun, safetypes.Some(notification.ActionUnlockRepository))
			if saveErr != nil {
				b.log.Error(fmt.Sprintf("Failed to save notification: %s", saveErr))
			}
			b.state.SetBackupError(b.ctx, bId, err, false, true)
			b.state.AddNotification(b.ctx, fmt.Sprintf("Backup job failed: repository %s is locked", repo.Name), types.LevelError)
			return BackupResultError, err
		} else {
			saveErr := b.saveDbNotification(bId, err.Error(), notification.TypeFailedBackupRun, safetypes.None[notification.Action]())
			if saveErr != nil {
				b.log.Error(fmt.Sprintf("Failed to save notification: %s", saveErr))
			}
			b.state.SetBackupError(b.ctx, bId, err, true, false)
			b.state.AddNotification(b.ctx, fmt.Sprintf("Backup job failed: %s", err), types.LevelError)
			return BackupResultError, err
		}
	} else {
		// Backup completed successfully
		defer b.state.SetBackupCompleted(b.ctx, bId, true)

		err = b.refreshRepoInfo(bId.RepositoryId, repo.Location, repo.Password)
		if err != nil {
			b.log.Error(fmt.Sprintf("Failed to get info for backup %d: %s", bId, err))
		}

		err = b.addNewArchive(bId, archiveName, repo.Password)
		if err != nil {
			b.log.Error(fmt.Sprintf("Failed to add new archive for backup %d: %s", bId, err))
		}

		pruningRule, err := b.db.PruningRule.
			Query().
			Where(pruningrule.And(
				pruningrule.HasBackupProfileWith(backupprofile.ID(bId.BackupProfileId)),
				pruningrule.IsEnabled(true),
			)).
			Only(b.ctx)
		if err != nil && !ent.IsNotFound(err) {
			b.log.Error(fmt.Sprintf("Failed to get pruning rule: %s", err))
		}
		if pruningRule != nil && pruningRule.IsEnabled {
			_, err := b.examinePrune(bId, safetypes.Some(pruningRule), true, true)
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

	err := b.borg.DeleteArchives(b.ctx, location, password, prefix)
	if err != nil {
		if errors.As(err, &borg.CancelErr{}) {
			b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusIdle)
			return DeleteResultCancelled, nil
		} else if errors.As(err, &borg.LockTimeout{}) {
			b.state.AddNotification(b.ctx, "Delete job failed: repository is locked", types.LevelError)
			b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusLocked)
			return DeleteResultError, err
		} else {
			b.state.AddNotification(b.ctx, fmt.Sprintf("Delete job failed: %s", err), types.LevelError)
			b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusIdle)
			return DeleteResultError, err
		}
	} else {
		// Delete completed successfully
		defer b.state.SetRepoStatus(b.ctx, bId.RepositoryId, state.RepoStatusIdle)

		err = b.refreshRepoInfo(bId.RepositoryId, location, password)
		if err != nil {
			b.log.Error(fmt.Sprintf("Failed to get info for backup-profile %d: %s", bId, err))
		}

		_, err = b.repoClient().refreshArchives(bId.RepositoryId)
		if err != nil {
			b.log.Error(fmt.Sprintf("Failed to refresh archives for backup-profile %d: %s", bId, err))
		}

		return DeleteResultSuccess, nil
	}
}

func (b *BackupClient) saveProgressInfo(id types.BackupId, ch chan borg.BackupProgress) {
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

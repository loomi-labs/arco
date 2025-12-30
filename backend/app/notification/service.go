package notification

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/notification"
	"go.uber.org/zap"
)

// ErrorNotification represents an error notification with its related entities
type ErrorNotification struct {
	ID                int       `json:"id"`
	Message           string    `json:"message"`
	Type              string    `json:"type"`
	CreatedAt         time.Time `json:"createdAt"`
	BackupProfileID   int       `json:"backupProfileId"`
	BackupProfileName string    `json:"backupProfileName"`
	RepositoryID      int       `json:"repositoryId"`
	RepositoryName    string    `json:"repositoryName"`
}

// ErrorCounts holds error counts per entity
type ErrorCounts struct {
	ByRepository    map[int]int `json:"byRepository"`
	ByBackupProfile map[int]int `json:"byBackupProfile"`
}

// Service contains the business logic for notifications
type Service struct {
	log          *zap.SugaredLogger
	db           *ent.Client
	eventEmitter types.EventEmitter
}

// NewService creates a new notification service
func NewService(log *zap.SugaredLogger) *Service {
	return &Service{
		log: log,
	}
}

// Init initializes the service with database client and event emitter
func (s *Service) Init(db *ent.Client, eventEmitter types.EventEmitter) {
	s.db = db
	s.eventEmitter = eventEmitter
}

// errorTypes are the notification types considered errors (not warnings)
var errorTypes = []notification.Type{
	notification.TypeFailedBackupRun,
	notification.TypeFailedPruningRun,
	notification.TypeFailedQuickCheck,
	notification.TypeFailedFullCheck,
}

// GetUnseenErrors returns all unseen error notifications with their related entities
func (s *Service) GetUnseenErrors(ctx context.Context) ([]ErrorNotification, error) {
	notifications, err := s.db.Notification.Query().
		Where(
			notification.SeenEQ(false),
			notification.TypeIn(errorTypes...),
		).
		WithBackupProfile().
		WithRepository().
		Order(notification.ByCreatedAt(sql.OrderDesc())).
		All(ctx)
	if err != nil {
		s.log.Errorf("Failed to get unseen errors: %v", err)
		return nil, err
	}

	result := make([]ErrorNotification, len(notifications))
	for i, n := range notifications {
		var profileName, repoName string
		var profileID, repoID int

		if n.Edges.BackupProfile != nil {
			profileID = n.Edges.BackupProfile.ID
			profileName = n.Edges.BackupProfile.Name
		}
		if n.Edges.Repository != nil {
			repoID = n.Edges.Repository.ID
			repoName = n.Edges.Repository.Name
		}

		result[i] = ErrorNotification{
			ID:                n.ID,
			Message:           n.Message,
			Type:              string(n.Type),
			CreatedAt:         n.CreatedAt,
			BackupProfileID:   profileID,
			BackupProfileName: profileName,
			RepositoryID:      repoID,
			RepositoryName:    repoName,
		}
	}
	return result, nil
}

// DismissError marks a notification as seen (dismissed)
func (s *Service) DismissError(ctx context.Context, id int) error {
	_, err := s.db.Notification.UpdateOneID(id).
		SetSeen(true).
		Save(ctx)
	if err != nil {
		s.log.Errorf("Failed to dismiss error %d: %v", id, err)
		return err
	}

	// Emit event to notify frontend
	s.eventEmitter.EmitEvent(ctx, types.EventNotificationDismissedString())
	return nil
}

// DismissAllErrors marks all unseen error notifications as seen
func (s *Service) DismissAllErrors(ctx context.Context) error {
	_, err := s.db.Notification.Update().
		Where(
			notification.SeenEQ(false),
			notification.TypeIn(errorTypes...),
		).
		SetSeen(true).
		Save(ctx)
	if err != nil {
		s.log.Errorf("Failed to dismiss all errors: %v", err)
		return err
	}

	s.eventEmitter.EmitEvent(ctx, types.EventNotificationDismissedString())
	return nil
}

// DismissErrors marks specific error notifications as seen by their IDs
func (s *Service) DismissErrors(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	_, err := s.db.Notification.Update().
		Where(notification.IDIn(ids...)).
		SetSeen(true).
		Save(ctx)
	if err != nil {
		s.log.Errorf("Failed to dismiss errors %v: %v", ids, err)
		return err
	}

	s.eventEmitter.EmitEvent(ctx, types.EventNotificationDismissedString())
	return nil
}

// GetUnseenErrorCounts returns counts of unseen errors per repository and backup profile
func (s *Service) GetUnseenErrorCounts(ctx context.Context) (*ErrorCounts, error) {
	notifications, err := s.db.Notification.Query().
		Where(
			notification.SeenEQ(false),
			notification.TypeIn(errorTypes...),
		).
		WithBackupProfile().
		WithRepository().
		All(ctx)
	if err != nil {
		s.log.Errorf("Failed to get unseen error counts: %v", err)
		return nil, err
	}

	counts := &ErrorCounts{
		ByRepository:    make(map[int]int),
		ByBackupProfile: make(map[int]int),
	}

	for _, n := range notifications {
		if n.Edges.Repository != nil {
			counts.ByRepository[n.Edges.Repository.ID]++
		}
		if n.Edges.BackupProfile != nil {
			counts.ByBackupProfile[n.Edges.BackupProfile.ID]++
		}
	}

	return counts, nil
}

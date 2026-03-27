package feedback

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"connectrpc.com/connect"
	arcov1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/api/v1/arcov1connect"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/ent"
	"go.uber.org/zap"
)

// Service contains the business logic and provides methods exposed to the frontend
type Service struct {
	log       *zap.SugaredLogger
	db        *ent.Client
	state     *state.State
	rpcClient arcov1connect.FeedbackServiceClient
}

// ServiceInternal provides backend-only methods that should not be exposed to frontend
type ServiceInternal struct {
	*Service
	arcov1connect.UnimplementedFeedbackServiceHandler
}

// NewService creates a new feedback service
func NewService(log *zap.SugaredLogger, state *state.State) *ServiceInternal {
	return &ServiceInternal{
		Service: &Service{
			log:   log,
			state: state,
		},
	}
}

// Init initializes the service with database and RPC client
func (si *ServiceInternal) Init(db *ent.Client, rpcClient arcov1connect.FeedbackServiceClient) {
	si.db = db
	si.rpcClient = rpcClient
}

func (s *Service) mustHaveDB() {
	if s.db == nil {
		panic("FeedbackService: database client is nil")
	}
}

// SubmitFeedback sends user feedback to the cloud service
func (s *Service) SubmitFeedback(ctx context.Context, category string, rating int, message string, email string) error {
	var protoCategory arcov1.FeedbackCategory
	switch category {
	case "bug":
		protoCategory = arcov1.FeedbackCategory_FEEDBACK_CATEGORY_BUG
	case "feature":
		protoCategory = arcov1.FeedbackCategory_FEEDBACK_CATEGORY_FEATURE_REQUEST
	case "general":
		protoCategory = arcov1.FeedbackCategory_FEEDBACK_CATEGORY_GENERAL
	default:
		return fmt.Errorf("invalid feedback category %q", category)
	}

	if rating < 0 || rating > 5 {
		return fmt.Errorf("invalid feedback rating %d", rating)
	}

	req := connect.NewRequest(&arcov1.SubmitFeedbackRequest{
		Category:   protoCategory,
		Rating:     int32(rating),
		Message:    message,
		Email:      email,
		AppVersion: types.Version,
		OsInfo:     runtime.GOOS,
	})

	_, err := s.rpcClient.SubmitFeedback(ctx, req)
	if err != nil {
		s.log.Errorf("Failed to submit feedback: %v", err)
		return err
	}

	return nil
}

// ShouldShowFeedbackPopup checks if the feedback popup should be shown.
// Shows after 1 month of usage, then once per year (on dismiss or submit).
func (s *Service) ShouldShowFeedbackPopup(ctx context.Context) (bool, error) {
	s.mustHaveDB()

	settings, err := s.db.Settings.Query().First(ctx)
	if err != nil {
		return false, err
	}

	// Must have used the app for at least 1 month
	if time.Since(settings.CreatedAt) < 30*24*time.Hour {
		return false, nil
	}

	// Never prompted before
	if settings.FeedbackLastPromptedAt == nil {
		return true, nil
	}

	// Prompted more than 1 year ago
	return time.Since(*settings.FeedbackLastPromptedAt) > 365*24*time.Hour, nil
}

// MarkFeedbackPrompted records that the feedback popup was shown (dismissed or submitted)
func (s *Service) MarkFeedbackPrompted(ctx context.Context) error {
	s.mustHaveDB()

	now := time.Now()
	return s.db.Settings.
		Update().
		SetFeedbackLastPromptedAt(now).
		Exec(ctx)
}

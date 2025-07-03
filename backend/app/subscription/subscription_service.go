package subscription

import (
	"connectrpc.com/connect"
	"context"
	arcov1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/api/v1/arcov1connect"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/pkg/browser"
	"go.uber.org/zap"
	"time"
)

// Service contains the business logic and provides methods exposed to the frontend
type Service struct {
	log       *zap.SugaredLogger
	db        *ent.Client
	state     *state.State
	rpcClient arcov1connect.SubscriptionServiceClient
}

// ServiceInternal provides backend-only methods that should not be exposed to frontend
type ServiceInternal struct {
	*Service
	arcov1connect.UnimplementedSubscriptionServiceHandler
}

// NewService creates a new subscription service
func NewService(log *zap.SugaredLogger, state *state.State) *ServiceInternal {
	return &ServiceInternal{
		Service: &Service{
			log:   log,
			state: state,
		},
	}
}

// Init initializes the service with database and RPC client
func (si *ServiceInternal) Init(db *ent.Client, rpcClient arcov1connect.SubscriptionServiceClient) {
	si.db = db
	si.rpcClient = rpcClient
}

// mustHaveDB panics if db is nil. This is a programming error guard.
func (s *Service) mustHaveDB() {
	if s.db == nil {
		panic("SubscriptionService: database client is nil")
	}
}

// Frontend-exposed business logic methods

// GetSubscription returns the user's current subscription
func (s *Service) GetSubscription(ctx context.Context, userID string) (*arcov1.GetSubscriptionResponse, error) {
	req := connect.NewRequest(&arcov1.GetSubscriptionRequest{
		UserId: userID,
	})

	resp, err := s.rpcClient.GetSubscription(ctx, req)
	if err != nil {
		s.log.Errorf("Failed to get subscription from cloud service: %v", err)
		return nil, err
	}

	return resp.Msg, nil
}

// CreateCheckoutSession creates a payment checkout session
func (s *Service) CreateCheckoutSession(ctx context.Context, planName string) (*arcov1.CreateCheckoutSessionResponse, error) {
	req := connect.NewRequest(&arcov1.CreateCheckoutSessionRequest{
		Name:     planName,
		Currency: arcov1.Currency_CURRENCY_CHF,
	})

	resp, err := s.rpcClient.CreateCheckoutSession(ctx, req)
	if err != nil {
		s.log.Errorf("Failed to create checkout session from cloud service: %v", err)
		return nil, err
	}

	// Store checkout session in state
	s.state.SetCheckoutSession(ctx, resp.Msg)

	// Start monitoring checkout completion in the background
	internal := &ServiceInternal{Service: s}
	go internal.startCheckoutMonitoring(resp.Msg.SessionId)

	// Open URL to complete the checkout
	err = browser.OpenURL(resp.Msg.CheckoutUrl)
	if err != nil {
		s.log.Errorf("Failed to open browser url: %v", err)
		return nil, err
	}

	return resp.Msg, nil
}

// CancelSubscription cancels the user's subscription
func (s *Service) CancelSubscription(ctx context.Context, subscriptionID string) (*arcov1.CancelSubscriptionResponse, error) {
	req := connect.NewRequest(&arcov1.CancelSubscriptionRequest{
		SubscriptionId: subscriptionID,
	})

	resp, err := s.rpcClient.CancelSubscription(ctx, req)
	if err != nil {
		s.log.Errorf("Failed to cancel subscription from cloud service: %v", err)
		return nil, err
	}

	return resp.Msg, nil
}

// ChangeBillingCycle schedules a billing cycle change for the next billing period
// This method now uses ScheduleSubscriptionUpdate instead of the deprecated ChangeBillingCycle RPC
func (s *Service) ChangeBillingCycle(ctx context.Context, subscriptionID string, isYearly bool) (*arcov1.ScheduleSubscriptionUpdateResponse, error) {
	req := connect.NewRequest(&arcov1.ScheduleSubscriptionUpdateRequest{
		SubscriptionId: subscriptionID,
		Change: &arcov1.ScheduleSubscriptionUpdateRequest_IsYearlyBilling{
			IsYearlyBilling: isYearly,
		},
	})

	resp, err := s.rpcClient.ScheduleSubscriptionUpdate(ctx, req)
	if err != nil {
		s.log.Errorf("Failed to schedule billing cycle change from cloud service: %v", err)
		return nil, err
	}

	return resp.Msg, nil
}

// ReactivateSubscription reactivates a cancelled subscription
func (s *Service) ReactivateSubscription(ctx context.Context, subscriptionID string) (*arcov1.ReactivateSubscriptionResponse, error) {
	req := connect.NewRequest(&arcov1.ReactivateSubscriptionRequest{
		SubscriptionId: subscriptionID,
	})

	resp, err := s.rpcClient.ReactivateSubscription(ctx, req)
	if err != nil {
		s.log.Errorf("Failed to reactivate subscription from cloud service: %v", err)
		return nil, err
	}

	return resp.Msg, nil
}

// GetCheckoutSession returns the current checkout session
func (s *Service) GetCheckoutSession() *arcov1.CreateCheckoutSessionResponse {
	return s.state.GetCheckoutSession()
}

// GetCheckoutResult returns the current checkout result
func (s *Service) GetCheckoutResult() *state.CheckoutResult {
	return s.state.GetCheckoutResult()
}

// ClearCheckoutResult clears the current checkout result
func (s *Service) ClearCheckoutResult() {
	s.state.ClearCheckoutResult()
}

// UpgradeSubscription performs immediate Basic→Pro plan upgrade with proration
func (s *Service) UpgradeSubscription(ctx context.Context, subscriptionID string, planID string) (*arcov1.UpgradeSubscriptionResponse, error) {
	req := connect.NewRequest(&arcov1.UpgradeSubscriptionRequest{
		SubscriptionId: subscriptionID,
		PlanId:         planID,
	})

	resp, err := s.rpcClient.UpgradeSubscription(ctx, req)
	if err != nil {
		s.log.Errorf("Failed to upgrade subscription from cloud service: %v", err)
		return nil, err
	}

	return resp.Msg, nil
}

// ScheduleSubscriptionUpdate schedules changes to take effect at the next billing cycle
func (s *Service) ScheduleSubscriptionUpdate(ctx context.Context, subscriptionID string, planID *string, currency *arcov1.Currency, isYearlyBilling *bool) (*arcov1.ScheduleSubscriptionUpdateResponse, error) {
	req := &arcov1.ScheduleSubscriptionUpdateRequest{
		SubscriptionId: subscriptionID,
	}

	// Set the appropriate change type based on what's provided
	if planID != nil {
		req.Change = &arcov1.ScheduleSubscriptionUpdateRequest_PlanId{
			PlanId: *planID,
		}
	} else if currency != nil {
		req.Change = &arcov1.ScheduleSubscriptionUpdateRequest_Currency{
			Currency: *currency,
		}
	} else if isYearlyBilling != nil {
		req.Change = &arcov1.ScheduleSubscriptionUpdateRequest_IsYearlyBilling{
			IsYearlyBilling: *isYearlyBilling,
		}
	}

	resp, err := s.rpcClient.ScheduleSubscriptionUpdate(ctx, connect.NewRequest(req))
	if err != nil {
		s.log.Errorf("Failed to schedule subscription update from cloud service: %v", err)
		return nil, err
	}

	return resp.Msg, nil
}

// GetPendingChanges retrieves all scheduled changes for a subscription
func (s *Service) GetPendingChanges(ctx context.Context, subscriptionID string) (*arcov1.GetPendingChangesResponse, error) {
	req := connect.NewRequest(&arcov1.GetPendingChangesRequest{
		SubscriptionId: subscriptionID,
	})

	resp, err := s.rpcClient.GetPendingChanges(ctx, req)
	if err != nil {
		s.log.Errorf("Failed to get pending changes from cloud service: %v", err)
		return nil, err
	}

	return resp.Msg, nil
}

// CancelPendingChange cancels a specific scheduled change before it takes effect
func (s *Service) CancelPendingChange(ctx context.Context, subscriptionID string, changeID int64) (*arcov1.CancelPendingChangeResponse, error) {
	req := connect.NewRequest(&arcov1.CancelPendingChangeRequest{
		SubscriptionId: subscriptionID,
		ChangeId:       changeID,
	})

	resp, err := s.rpcClient.CancelPendingChange(ctx, req)
	if err != nil {
		s.log.Errorf("Failed to cancel pending change from cloud service: %v", err)
		return nil, err
	}

	return resp.Msg, nil
}

// Backend-only Connect RPC handler methods

// startCheckoutMonitoring starts monitoring a checkout session for completion
func (si *ServiceInternal) startCheckoutMonitoring(sessionId string) {
	// Create a timeout context with reasonable timeout for checkout completion
	// Most payment flows should complete within 30 minutes
	const checkoutTimeout = 30 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), checkoutTimeout)
	defer cancel()

	// Retry configuration
	const maxRetries = 120 // Max 120 retries over 10 minutes
	const retryInterval = 5 * time.Second

	// Helper function to create checkout result
	createResult := func(status state.CheckoutResultStatus, errorMessage string) *state.CheckoutResult {
		return &state.CheckoutResult{
			Status:       status,
			ErrorMessage: errorMessage,
		}
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Use WaitForCheckoutCompletion streaming approach
		req := connect.NewRequest(&arcov1.WaitForCheckoutCompletionRequest{SessionId: sessionId})

		si.log.Debugf("Checkout session %s: attempting checkout stream (attempt %d/%d)", sessionId, attempt+1, maxRetries)
		stream, err := si.rpcClient.WaitForCheckoutCompletion(ctx, req)
		if err != nil {
			si.log.Debugf("Checkout session %s: stream connection failed: %v", sessionId, err)
			if attempt == maxRetries-1 {
				si.log.Debugf("Checkout session %s: max retries reached, stopping monitoring", sessionId)
				si.state.ClearCheckoutSession(ctx, createResult(state.CheckoutStatusTimeout, "Checkout monitoring timed out"), false)
				return
			}

			// Wait before retry
			select {
			case <-ctx.Done():
				si.state.ClearCheckoutSession(ctx, createResult(state.CheckoutStatusTimeout, "Checkout monitoring timed out"), false)
				return
			case <-time.After(retryInterval):
				continue
			}
		}

		// Stream established successfully
		si.log.Debugf("Checkout session %s: checkout stream established", sessionId)

		for stream.Receive() {
			checkoutStatus := stream.Msg()

			switch checkoutStatus.Status {
			case arcov1.CheckoutStatus_CHECKOUT_STATUS_COMPLETED:
				// Checkout successful
				si.log.Debugf("Checkout session %s: checkout completed successfully", sessionId)
				result := createResult(state.CheckoutStatusCompleted, "")
				result.SubscriptionID = checkoutStatus.SubscriptionId
				// Clear checkout session from state and emit both checkout and subscription events
				si.state.ClearCheckoutSession(ctx, result, true)
				return
			case arcov1.CheckoutStatus_CHECKOUT_STATUS_FAILED, arcov1.CheckoutStatus_CHECKOUT_STATUS_EXPIRED:
				// Checkout failed
				si.log.Debugf("Checkout session %s: checkout failed with status %v", sessionId, checkoutStatus.Status)
				result := createResult(state.CheckoutStatusFailed, "Checkout failed or expired")
				// Clear checkout session from state (automatically emits checkout event only)
				si.state.ClearCheckoutSession(ctx, result, false)
				return
			case arcov1.CheckoutStatus_CHECKOUT_STATUS_PENDING:
				// Still pending - continue waiting
				si.log.Debugf("Checkout session %s: pending checkout", sessionId)
				continue
			default:
				// Unknown status
				si.log.Debugf("Checkout session %s: unknown checkout status %v", sessionId, checkoutStatus.Status)
				result := createResult(state.CheckoutStatusFailed, "Unknown checkout status")
				// Clear checkout session from state (automatically emits checkout event only)
				si.state.ClearCheckoutSession(ctx, result, false)
				return
			}
		}

		// Stream ended - check for errors and retry if not max attempts
		if err := stream.Err(); err != nil {
			si.log.Debugf("Checkout session %s: stream error: %v", sessionId, err)
			if attempt == maxRetries-1 {
				si.log.Debugf("Checkout session %s: max retries reached after error", sessionId)
				si.state.ClearCheckoutSession(ctx, createResult(state.CheckoutStatusTimeout, "Checkout monitoring timed out"), false)
				return
			}

			// Wait before retry
			select {
			case <-ctx.Done():
				si.state.ClearCheckoutSession(ctx, createResult(state.CheckoutStatusTimeout, "Checkout monitoring timed out"), false)
				return
			case <-time.After(retryInterval):
				continue
			}
		}

		// Stream ended without error - retry
		si.log.Debugf("Checkout session %s: stream ended, retrying", sessionId)
	}

	// If we reach here, max retries were exhausted
	si.log.Debugf("Checkout session %s: monitoring timed out after %d attempts", sessionId, maxRetries)
	si.state.ClearCheckoutSession(ctx, createResult(state.CheckoutStatusTimeout, "Checkout monitoring timed out"), false)
}

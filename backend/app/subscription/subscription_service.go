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
		Name: planName,
	})

	resp, err := s.rpcClient.CreateCheckoutSession(ctx, req)
	if err != nil {
		s.log.Errorf("Failed to create checkout session from cloud service: %v", err)
		return nil, err
	}

	// Store checkout session in state
	s.state.SetCheckoutSession(ctx, resp.Msg, false)

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
func (s *Service) CancelSubscription(ctx context.Context, subscriptionID string, immediate bool) (*arcov1.CancelSubscriptionResponse, error) {
	req := connect.NewRequest(&arcov1.CancelSubscriptionRequest{
		SubscriptionId: subscriptionID,
		Immediate:      immediate,
	})

	resp, err := s.rpcClient.CancelSubscription(ctx, req)
	if err != nil {
		s.log.Errorf("Failed to cancel subscription from cloud service: %v", err)
		return nil, err
	}

	return resp.Msg, nil
}

// GetCheckoutSession returns the current checkout session
func (s *Service) GetCheckoutSession() *arcov1.CreateCheckoutSessionResponse {
	return s.state.GetCheckoutSession()
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

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Use WaitForCheckoutCompletion streaming approach
		req := connect.NewRequest(&arcov1.WaitForCheckoutCompletionRequest{SessionId: sessionId})

		si.log.Debugf("Checkout session %s: attempting checkout stream (attempt %d/%d)", sessionId, attempt+1, maxRetries)
		stream, err := si.rpcClient.WaitForCheckoutCompletion(ctx, req)
		if err != nil {
			si.log.Debugf("Checkout session %s: stream connection failed: %v", sessionId, err)
			if attempt == maxRetries-1 {
				si.log.Debugf("Checkout session %s: max retries reached, stopping monitoring", sessionId)
				return
			}

			// Wait before retry
			select {
			case <-ctx.Done():
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
				// Clear checkout session from state and emit both checkout and subscription events
				si.state.ClearCheckoutSession(ctx, true)
				return
			case arcov1.CheckoutStatus_CHECKOUT_STATUS_FAILED, arcov1.CheckoutStatus_CHECKOUT_STATUS_EXPIRED:
				// Checkout failed
				si.log.Debugf("Checkout session %s: checkout failed with status %v", sessionId, checkoutStatus.Status)
				// Clear checkout session from state (automatically emits checkout event only)
				si.state.ClearCheckoutSession(ctx, false)
				return
			case arcov1.CheckoutStatus_CHECKOUT_STATUS_PENDING:
				// Still pending - continue waiting
				si.log.Debugf("Checkout session %s: pending checkout", sessionId)
				continue
			default:
				// Unknown status
				si.log.Debugf("Checkout session %s: unknown checkout status %v", sessionId, checkoutStatus.Status)
				return
			}
		}

		// Stream ended - check for errors and retry if not max attempts
		if err := stream.Err(); err != nil {
			si.log.Debugf("Checkout session %s: stream error: %v", sessionId, err)
			if attempt == maxRetries-1 {
				si.log.Debugf("Checkout session %s: max retries reached after error", sessionId)
				return
			}

			// Wait before retry
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryInterval):
				continue
			}
		}

		// Stream ended without error - retry
		si.log.Debugf("Checkout session %s: stream ended, retrying", sessionId)
	}

	si.log.Debugf("Checkout session %s: monitoring completed", sessionId)
}

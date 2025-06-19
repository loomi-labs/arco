package subscription

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"
	arcov1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/api/v1/arcov1connect"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/ent"
	"go.uber.org/zap"
)

// Service contains the business logic and provides methods exposed to the frontend
type Service struct {
	log       *zap.SugaredLogger
	db        *ent.Client
	state     *state.State
	rpcClient arcov1connect.SubscriptionServiceClient
}

// ServiceInternal provides backend-only methods and implements Connect RPC handlers
type ServiceInternal struct {
	*Service
	arcov1connect.UnimplementedSubscriptionServiceHandler
}

// NewService creates a new subscription service
func NewService(log *zap.SugaredLogger, state *state.State, cloudRPCURL string) *ServiceInternal {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &ServiceInternal{
		Service: &Service{
			log:       log,
			state:     state,
			rpcClient: arcov1connect.NewSubscriptionServiceClient(httpClient, cloudRPCURL),
		},
	}
}

func (s *Service) SetDb(db *ent.Client) {
	s.db = db
}

// mustHaveDB panics if db is nil. This is a programming error guard.
func (s *Service) mustHaveDB() {
	if s.db == nil {
		panic("SubscriptionService: database client is nil")
	}
}

// Frontend-exposed business logic methods

// ListPlans returns available subscription plans and regions
func (s *Service) ListPlans(ctx context.Context) (*arcov1.ListPlansResponse, error) {
	// Mock data for now - will be replaced with real payment provider integration
	plans := []*arcov1.Plan{
		{
			Name:              "Basic",
			FeatureSet:        arcov1.FeatureSet_BASIC,
			PriceMonthlyCents: 500,  // $5.00
			PriceYearlyCents:  5000, // $50.00
			Currency:          "USD",
			StorageGb:         250,
			HasFreeTrial:      true,
		},
		{
			Name:              "Pro",
			FeatureSet:        arcov1.FeatureSet_PRO,
			PriceMonthlyCents: 1200,  // $12.00
			PriceYearlyCents:  12000, // $120.00
			Currency:          "USD",
			StorageGb:         1000, // 1TB
			HasFreeTrial:      true,
		},
	}

	regions := []string{"Europe", "US"}

	return &arcov1.ListPlansResponse{
		Plans:   plans,
		Regions: regions,
	}, nil
}

// GetSubscription returns the user's current subscription
func (s *Service) GetSubscription(ctx context.Context, userID string) (*arcov1.GetSubscriptionResponse, error) {
	// Mock response - will be replaced with real database lookup
	return &arcov1.GetSubscriptionResponse{
		Subscription: nil, // No active subscription for now
	}, nil
}

// CreateCheckoutSession creates a payment checkout session
func (s *Service) CreateCheckoutSession(ctx context.Context, planID, successURL, cancelURL string) (*arcov1.CreateCheckoutSessionResponse, error) {
	// Mock response - will be replaced with real payment provider integration
	return &arcov1.CreateCheckoutSessionResponse{
		SessionId:   "mock_session_id",
		CheckoutUrl: "https://mock-checkout-url.com",
	}, nil
}

// CancelSubscription cancels the user's subscription
func (s *Service) CancelSubscription(ctx context.Context, subscriptionID string, immediate bool) (*arcov1.CancelSubscriptionResponse, error) {
	// Mock response - will be replaced with real payment provider integration
	return &arcov1.CancelSubscriptionResponse{
		Success: true,
		Message: "Subscription will be canceled at the end of the billing period",
	}, nil
}

// Backend-only Connect RPC handler methods

// ListPlans handles the Connect RPC request for listing plans
func (si *ServiceInternal) ListPlans(ctx context.Context, req *connect.Request[arcov1.ListPlansRequest]) (*connect.Response[arcov1.ListPlansResponse], error) {
	resp, err := si.Service.ListPlans(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// GetSubscription handles the Connect RPC request for getting a subscription
func (si *ServiceInternal) GetSubscription(ctx context.Context, req *connect.Request[arcov1.GetSubscriptionRequest]) (*connect.Response[arcov1.GetSubscriptionResponse], error) {
	resp, err := si.Service.GetSubscription(ctx, req.Msg.UserId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// CreateCheckoutSession handles the Connect RPC request for creating a checkout session
func (si *ServiceInternal) CreateCheckoutSession(ctx context.Context, req *connect.Request[arcov1.CreateCheckoutSessionRequest]) (*connect.Response[arcov1.CreateCheckoutSessionResponse], error) {
	resp, err := si.Service.CreateCheckoutSession(ctx, req.Msg.PlanId, req.Msg.SuccessUrl, req.Msg.CancelUrl)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// CancelSubscription handles the Connect RPC request for canceling a subscription
func (si *ServiceInternal) CancelSubscription(ctx context.Context, req *connect.Request[arcov1.CancelSubscriptionRequest]) (*connect.Response[arcov1.CancelSubscriptionResponse], error) {
	resp, err := si.Service.CancelSubscription(ctx, req.Msg.SubscriptionId, req.Msg.Immediate)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

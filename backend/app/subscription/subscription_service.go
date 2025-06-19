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

// ServiceRPC implements Connect RPC handlers for the subscription service
type ServiceRPC struct {
	*Service
	arcov1connect.UnimplementedSubscriptionServiceHandler
}

// NewService creates a new subscription service
func NewService(log *zap.SugaredLogger, state *state.State, cloudRPCURL string) *ServiceRPC {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &ServiceRPC{
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
func (s *Service) CreateCheckoutSession(ctx context.Context, planID, successURL, cancelURL string) (*arcov1.CreateCheckoutSessionResponse, error) {
	req := connect.NewRequest(&arcov1.CreateCheckoutSessionRequest{
		PlanId:     planID,
		SuccessUrl: successURL,
		CancelUrl:  cancelURL,
	})
	
	resp, err := s.rpcClient.CreateCheckoutSession(ctx, req)
	if err != nil {
		s.log.Errorf("Failed to create checkout session from cloud service: %v", err)
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

// Backend-only Connect RPC handler methods

// GetSubscription handles the Connect RPC request for getting a subscription
func (si *ServiceRPC) GetSubscription(ctx context.Context, req *connect.Request[arcov1.GetSubscriptionRequest]) (*connect.Response[arcov1.GetSubscriptionResponse], error) {
	resp, err := si.Service.GetSubscription(ctx, req.Msg.UserId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// CreateCheckoutSession handles the Connect RPC request for creating a checkout session
func (si *ServiceRPC) CreateCheckoutSession(ctx context.Context, req *connect.Request[arcov1.CreateCheckoutSessionRequest]) (*connect.Response[arcov1.CreateCheckoutSessionResponse], error) {
	resp, err := si.Service.CreateCheckoutSession(ctx, req.Msg.PlanId, req.Msg.SuccessUrl, req.Msg.CancelUrl)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// CancelSubscription handles the Connect RPC request for canceling a subscription
func (si *ServiceRPC) CancelSubscription(ctx context.Context, req *connect.Request[arcov1.CancelSubscriptionRequest]) (*connect.Response[arcov1.CancelSubscriptionResponse], error) {
	resp, err := si.Service.CancelSubscription(ctx, req.Msg.SubscriptionId, req.Msg.Immediate)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

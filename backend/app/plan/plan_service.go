package plan

import (
	"context"

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
	rpcClient arcov1connect.PlanServiceClient
}

// ServiceRPC implements Connect RPC handlers for the plan service
type ServiceRPC struct {
	*Service
	arcov1connect.UnimplementedPlanServiceHandler
}

// NewService creates a new plan service
func NewService(log *zap.SugaredLogger, state *state.State) *ServiceRPC {
	return &ServiceRPC{
		Service: &Service{
			log:   log,
			state: state,
		},
	}
}

// Init initializes the service with database and RPC client
func (s *Service) Init(db *ent.Client, rpcClient arcov1connect.PlanServiceClient) {
	s.db = db
	s.rpcClient = rpcClient
}

// mustHaveDB panics if db is nil. This is a programming error guard.
func (s *Service) mustHaveDB() {
	if s.db == nil {
		panic("PlanService: database client is nil")
	}
}

// Frontend-exposed business logic methods

// ListPlans returns available subscription plans
func (s *Service) ListPlans(ctx context.Context) (*arcov1.ListPlansResponse, error) {
	req := connect.NewRequest(&arcov1.ListPlansRequest{})

	resp, err := s.rpcClient.ListPlans(ctx, req)
	if err != nil {
		s.log.Errorf("Failed to list plans from cloud service: %v", err)
		return nil, err
	}

	return resp.Msg, nil
}

// Backend-only Connect RPC handler methods

// ListPlans handles the Connect RPC request for listing plans
func (si *ServiceRPC) ListPlans(ctx context.Context, req *connect.Request[arcov1.ListPlansRequest]) (*connect.Response[arcov1.ListPlansResponse], error) {
	resp, err := si.Service.ListPlans(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

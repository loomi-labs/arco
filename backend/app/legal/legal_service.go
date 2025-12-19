package legal

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
	rpcClient arcov1connect.LegalServiceClient
}

// ServiceInternal provides backend-only methods that should not be exposed to frontend
type ServiceInternal struct {
	*Service
	arcov1connect.UnimplementedLegalServiceHandler
}

// NewService creates a new legal service
func NewService(log *zap.SugaredLogger, state *state.State) *ServiceInternal {
	return &ServiceInternal{
		Service: &Service{
			log:   log,
			state: state,
		},
	}
}

// Init initializes the service with database and RPC client
func (si *ServiceInternal) Init(db *ent.Client, rpcClient arcov1connect.LegalServiceClient) {
	si.db = db
	si.rpcClient = rpcClient
}

// Frontend-exposed business logic methods

// GetLegalDocuments returns the current terms of service and privacy policy
func (s *Service) GetLegalDocuments(ctx context.Context) (*arcov1.GetLegalDocumentsResponse, error) {
	req := connect.NewRequest(&arcov1.GetLegalDocumentsRequest{})

	resp, err := s.rpcClient.GetLegalDocuments(ctx, req)
	if err != nil {
		s.log.Errorf("Failed to get legal documents from cloud service: %v", err)
		return nil, err
	}

	return resp.Msg, nil
}

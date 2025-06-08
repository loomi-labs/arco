package auth

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/api/v1/arcov1connect"
	"go.uber.org/zap"
)

type AuthServiceHandler struct {
	log       *zap.SugaredLogger
	RpcClient arcov1connect.AuthServiceClient
}

func NewAuthServiceHandler(log *zap.SugaredLogger, cloudRPCURL string) *AuthServiceHandler {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	rpcClient := arcov1connect.NewAuthServiceClient(httpClient, cloudRPCURL)

	return &AuthServiceHandler{
		log:       log,
		RpcClient: rpcClient,
	}
}

func (h *AuthServiceHandler) Register(ctx context.Context, req *connect.Request[v1.RegisterRequest]) (*connect.Response[v1.RegisterResponse], error) {
	return h.RpcClient.Register(ctx, req)
}

func (h *AuthServiceHandler) Login(ctx context.Context, req *connect.Request[v1.LoginRequest]) (*connect.Response[v1.LoginResponse], error) {
	return h.RpcClient.Login(ctx, req)
}

func (h *AuthServiceHandler) WaitForAuthentication(ctx context.Context, req *connect.Request[v1.WaitForAuthRequest], stream *connect.ServerStream[v1.AuthStatusResponse]) error {
	externalStream, err := h.RpcClient.WaitForAuthentication(ctx, req)
	if err != nil {
		return err
	}
	defer externalStream.Close()

	// Forward stream updates from external service to local client
	for externalStream.Receive() {
		resp := externalStream.Msg()

		// Send response to local client stream
		if err := stream.Send(resp); err != nil {
			return err
		}

		// Non-pending statuses end the stream
		if resp.Status != v1.AuthStatus_PENDING {
			return nil
		}
	}

	// Handle stream error
	if err := externalStream.Err(); err != nil {
		return err
	}

	return nil
}

func (h *AuthServiceHandler) RefreshToken(ctx context.Context, req *connect.Request[v1.RefreshTokenRequest]) (*connect.Response[v1.RefreshTokenResponse], error) {
	return h.RpcClient.RefreshToken(ctx, req)
}

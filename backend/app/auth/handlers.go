package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/api/v1/arcov1connect"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/authsession"
	"github.com/loomi-labs/arco/backend/ent/refreshtoken"
	"github.com/loomi-labs/arco/backend/ent/user"
)


type AuthServiceHandler struct {
	db         *ent.Client
	jwtService *JWTService
	RpcClient  arcov1connect.AuthServiceClient
}

func NewAuthServiceHandler(db *ent.Client, jwtService *JWTService, cloudRPCURL string) *AuthServiceHandler {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	rpcClient := arcov1connect.NewAuthServiceClient(httpClient, cloudRPCURL)

	return &AuthServiceHandler{
		db:         db,
		jwtService: jwtService,
		RpcClient:  rpcClient,
	}
}

func (h *AuthServiceHandler) Register(ctx context.Context, req *connect.Request[v1.RegisterRequest]) (*connect.Response[v1.RegisterResponse], error) {
	// Proxy request to external auth service
	resp, err := h.RpcClient.Register(ctx, req)
	if err != nil {
		return nil, err
	}

	// Store session locally for real-time updates
	sessionID := resp.Msg.SessionId
	email := req.Msg.Email

	// Use server-provided expiration time
	expiresAt := resp.Msg.ExpiresAt.AsTime()
	_, err = h.db.AuthSession.Create().
		SetID(sessionID).
		SetUserEmail(email).
		SetStatus(authsession.StatusPENDING).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create local auth session: %w", err))
	}

	return resp, nil
}

func (h *AuthServiceHandler) Login(ctx context.Context, req *connect.Request[v1.LoginRequest]) (*connect.Response[v1.LoginResponse], error) {
	// Proxy request to external auth service
	resp, err := h.RpcClient.Login(ctx, req)
	if err != nil {
		return nil, err
	}

	// Store session locally for real-time updates
	sessionID := resp.Msg.SessionId
	email := req.Msg.Email

	// Use server-provided expiration time
	expiresAt := resp.Msg.ExpiresAt.AsTime()
	_, err = h.db.AuthSession.Create().
		SetID(sessionID).
		SetUserEmail(email).
		SetStatus(authsession.StatusPENDING).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create local auth session: %w", err))
	}

	return resp, nil
}

func (h *AuthServiceHandler) WaitForAuthentication(ctx context.Context, req *connect.Request[v1.WaitForAuthRequest], stream *connect.ServerStream[v1.AuthStatusResponse]) error {
	sessionID := req.Msg.SessionId
	if sessionID == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("session_id is required"))
	}

	// Open stream to external service
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

		// If authenticated, sync tokens locally and end stream
		if resp.Status == v1.AuthStatus_AUTHENTICATED {
			h.syncAuthenticatedSession(ctx, sessionID, resp)
			return nil
		} else if resp.Status != v1.AuthStatus_PENDING {
			// Session expired or cancelled, update local session
			err = h.db.AuthSession.Update().
				Where(authsession.IDEQ(sessionID)).
				SetStatus(authsession.Status(resp.Status.String())).
				Exec(ctx)
			if err != nil {
				fmt.Printf("Warning: failed to update local session status: %v\n", err)
			}
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
	// Proxy to external service
	resp, err := h.RpcClient.RefreshToken(ctx, req)
	if err != nil {
		return nil, err
	}

	// Update local token storage
	refreshTokenString := req.Msg.RefreshToken
	tokenHash := h.jwtService.HashRefreshToken(refreshTokenString)

	storedToken, err := h.db.RefreshToken.Query().
		Where(refreshtoken.TokenHashEQ(tokenHash)).
		WithUser().
		First(ctx)
	if err == nil && storedToken != nil {
		newTokenHash := h.jwtService.HashRefreshToken(resp.Msg.RefreshToken)

		// Calculate expiration with 10% buffer for refresh token
		expiresIn := time.Duration(resp.Msg.ExpiresIn) * time.Second
		buffer := expiresIn / 10
		expiresAt := time.Now().Add(expiresIn - buffer)

		err = h.db.RefreshToken.Update().
			Where(refreshtoken.IDEQ(storedToken.ID)).
			SetTokenHash(newTokenHash).
			SetLastUsedAt(time.Now()).
			SetExpiresAt(expiresAt).
			Exec(ctx)
		if err != nil {
			// Log error but don't fail the request since external service succeeded
			fmt.Printf("Warning: failed to update local refresh token: %v\n", err)
		}
	}

	return resp, nil
}

// syncAuthenticatedSession syncs tokens from external service to local db
func (h *AuthServiceHandler) syncAuthenticatedSession(ctx context.Context, sessionID string, authResp *v1.AuthStatusResponse) {
	if authResp.User == nil {
		return
	}

	// Find or create user locally
	var userEntity *ent.User
	existingUser, err := h.db.User.Query().Where(user.EmailEQ(authResp.User.Email)).First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		fmt.Printf("Error checking existing user: %v\n", err)
		return
	}

	if existingUser != nil {
		userEntity = existingUser
		_, err = h.db.User.Update().
			Where(user.IDEQ(userEntity.ID)).
			SetLastLoggedIn(time.Now()).
			Save(ctx)
		if err != nil {
			fmt.Printf("Error updating user last login: %v\n", err)
		}
	} else {
		userEntity, err = h.db.User.Create().
			SetEmail(authResp.User.Email).
			SetLastLoggedIn(time.Now()).
			Save(ctx)
		if err != nil {
			fmt.Printf("Error creating user: %v\n", err)
			return
		}
	}

	// Store refresh token locally
	if authResp.RefreshToken != "" {
		refreshTokenHash := h.jwtService.HashRefreshToken(authResp.RefreshToken)

		// Calculate expiration with 10% buffer from server-provided refresh token expiration
		expiresIn := time.Duration(authResp.RefreshTokenExpiresIn) * time.Second
		buffer := expiresIn / 10
		expiresAt := time.Now().Add(expiresIn - buffer)

		_, err = h.db.RefreshToken.Create().
			SetUserID(userEntity.ID).
			SetTokenHash(refreshTokenHash).
			SetExpiresAt(expiresAt).
			Save(ctx)
		if err != nil {
			fmt.Printf("Error saving refresh token: %v\n", err)
		}
	}

	// Update local session
	err = h.db.AuthSession.Update().
		Where(authsession.IDEQ(sessionID)).
		SetStatus(authsession.StatusAUTHENTICATED).
		Exec(ctx)
	if err != nil {
		fmt.Printf("Error updating session status: %v\n", err)
	}
}

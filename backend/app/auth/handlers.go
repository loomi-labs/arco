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
	rpcClient  arcov1connect.AuthServiceClient
}

func NewAuthServiceHandler(db *ent.Client, jwtService *JWTService, cloudRPCURL string) *AuthServiceHandler {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	rpcClient := arcov1connect.NewAuthServiceClient(httpClient, cloudRPCURL)

	return &AuthServiceHandler{
		db:         db,
		jwtService: jwtService,
		rpcClient:  rpcClient,
	}
}

func (h *AuthServiceHandler) Register(ctx context.Context, req *connect.Request[v1.RegisterRequest]) (*connect.Response[v1.RegisterResponse], error) {
	// Proxy request to external auth service
	resp, err := h.rpcClient.Register(ctx, req)
	if err != nil {
		return nil, err
	}

	// Store session locally for real-time updates
	sessionID := resp.Msg.SessionId
	email := req.Msg.Email

	_, err = h.db.AuthSession.Create().
		SetID(sessionID).
		SetUserEmail(email).
		SetStatus(authsession.StatusPENDING).
		SetExpiresAt(time.Now().Add(10 * time.Minute)).
		Save(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create local auth session: %w", err))
	}

	return resp, nil
}

func (h *AuthServiceHandler) Login(ctx context.Context, req *connect.Request[v1.LoginRequest]) (*connect.Response[v1.LoginResponse], error) {
	// Proxy request to external auth service
	resp, err := h.rpcClient.Login(ctx, req)
	if err != nil {
		return nil, err
	}

	// Store session locally for real-time updates
	sessionID := resp.Msg.SessionId
	email := req.Msg.Email

	_, err = h.db.AuthSession.Create().
		SetID(sessionID).
		SetUserEmail(email).
		SetStatus(authsession.StatusPENDING).
		SetExpiresAt(time.Now().Add(10 * time.Minute)).
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

	// Poll external service and send updates to stream
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check external service status
			checkReq := connect.NewRequest(&v1.CheckAuthStatusRequest{SessionId: sessionID})
			resp, err := h.rpcClient.CheckAuthStatus(ctx, checkReq)
			if err != nil {
				continue
			}

			// Convert to AuthStatusResponse and send to stream
			authResponse := &v1.AuthStatusResponse{
				Status:       resp.Msg.Status,
				AccessToken:  resp.Msg.AccessToken,
				RefreshToken: resp.Msg.RefreshToken,
				ExpiresIn:    resp.Msg.ExpiresIn,
				User:         resp.Msg.User,
			}
			if err := stream.Send(authResponse); err != nil {
				return err
			}

			// If authenticated, sync tokens locally and end stream
			if resp.Msg.Status == v1.AuthStatus_AUTHENTICATED {
				h.syncAuthenticatedSession(ctx, sessionID, resp.Msg)
				return nil
			} else if resp.Msg.Status != v1.AuthStatus_PENDING {
				// Session expired or cancelled, update local session
				err = h.db.AuthSession.Update().
					Where(authsession.IDEQ(sessionID)).
					SetStatus(authsession.Status(resp.Msg.Status.String())).
					Exec(ctx)
				if err != nil {
					fmt.Printf("Warning: failed to update local session status: %v\n", err)
				}
				return nil
			}
		}
	}
}

func (h *AuthServiceHandler) CheckAuthStatus(ctx context.Context, req *connect.Request[v1.CheckAuthStatusRequest]) (*connect.Response[v1.CheckAuthStatusResponse], error) {
	// Check external service first
	resp, err := h.rpcClient.CheckAuthStatus(ctx, req)
	if err != nil {
		return nil, err
	}

	// If authenticated, sync tokens locally
	if resp.Msg.Status == v1.AuthStatus_AUTHENTICATED {
		h.syncAuthenticatedSession(ctx, req.Msg.SessionId, resp.Msg)
	}

	return resp, nil
}

func (h *AuthServiceHandler) RefreshToken(ctx context.Context, req *connect.Request[v1.RefreshTokenRequest]) (*connect.Response[v1.RefreshTokenResponse], error) {
	// Proxy to external service
	resp, err := h.rpcClient.RefreshToken(ctx, req)
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

		err = h.db.RefreshToken.Update().
			Where(refreshtoken.IDEQ(storedToken.ID)).
			SetTokenHash(newTokenHash).
			SetLastUsedAt(time.Now()).
			SetExpiresAt(time.Now().Add(360 * 24 * time.Hour)).
			Exec(ctx)
		if err != nil {
			// Log error but don't fail the request since external service succeeded
			fmt.Printf("Warning: failed to update local refresh token: %v\n", err)
		}
	}

	return resp, nil
}

// syncAuthenticatedSession syncs tokens from external service to local storage
func (h *AuthServiceHandler) syncAuthenticatedSession(ctx context.Context, sessionID string, authResp *v1.CheckAuthStatusResponse) {
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
		_, err = h.db.RefreshToken.Create().
			SetUserID(userEntity.ID).
			SetTokenHash(refreshTokenHash).
			SetExpiresAt(time.Now().Add(360 * 24 * time.Hour)).
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

func (h *AuthServiceHandler) CompleteAuthentication(ctx context.Context, sessionID string) error {
	// Check database session status
	session, err := h.db.AuthSession.Query().
		Where(authsession.IDEQ(sessionID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("session not found")
		}
		return fmt.Errorf("failed to query session: %w", err)
	}

	if session.Status == authsession.StatusAUTHENTICATED {
		return nil // Already completed
	}

	return fmt.Errorf("authentication not yet completed")
}

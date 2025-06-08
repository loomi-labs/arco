package app

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"time"

	v1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/app/auth"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/authsession"
	"github.com/loomi-labs/arco/backend/ent/user"
)

type AuthClient struct {
	app          *App
	eventEmitter types.EventEmitter
}

var (
	// Shared instance to maintain session state
	globalAuthHandler *auth.AuthServiceHandler
)

func (a *App) AuthClient() *AuthClient {
	return &AuthClient{
		app:          a,
		eventEmitter: &types.RuntimeEventEmitter{},
	}
}

func (ac *AuthClient) getAuthHandler() *auth.AuthServiceHandler {
	if globalAuthHandler == nil {
		if ac.app.db == nil {
			panic("database not initialized - this is a programming error")
		}
		globalAuthHandler = auth.NewAuthServiceHandler(ac.app.log, ac.app.config.CloudRPCURL)
	}
	return globalAuthHandler
}

func (ac *AuthClient) StartRegister(ctx context.Context, email string) (*v1.RegisterResponse, error) {
	authHandler := ac.getAuthHandler()

	req := connect.NewRequest(&v1.RegisterRequest{Email: email})

	// Make request to external auth service
	resp, err := authHandler.RpcClient.Register(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to register with external auth service: %w", err)
	}

	// Store session locally for real-time updates
	sessionID := resp.Msg.SessionId
	expiresAt := resp.Msg.ExpiresAt.AsTime()
	_, err = ac.app.db.AuthSession.Create().
		SetID(sessionID).
		SetStatus(authsession.StatusPENDING).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create local auth session: %w", err)
	}

	// Start monitoring authentication in the background
	go ac.startAuthMonitoring(context.Background(), resp.Msg.SessionId)

	return resp.Msg, nil
}

func (ac *AuthClient) StartLogin(ctx context.Context, email string) (*v1.LoginResponse, error) {
	authHandler := ac.getAuthHandler()

	req := connect.NewRequest(&v1.LoginRequest{Email: email})

	// Make request to external auth service
	resp, err := authHandler.RpcClient.Login(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to login with external auth service: %w", err)
	}

	// Store session locally for real-time updates
	sessionID := resp.Msg.SessionId
	expiresAt := resp.Msg.ExpiresAt.AsTime()
	_, err = ac.app.db.AuthSession.Create().
		SetID(sessionID).
		SetStatus(authsession.StatusPENDING).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create local auth session: %w", err)
	}

	// Start monitoring authentication in the background
	go ac.startAuthMonitoring(context.Background(), resp.Msg.SessionId)

	return resp.Msg, nil
}

func (ac *AuthClient) RefreshToken(ctx context.Context, refreshToken string) bool {
	authHandler := ac.getAuthHandler()

	// Update local token storage - get the first user since we only store one user
	userEntity, err := ac.app.db.User.Query().First(ctx)
	if err != nil {
		ac.app.log.Errorf("Failed to find user for token update: %v", err)
		return false
	}

	req := connect.NewRequest(&v1.RefreshTokenRequest{RefreshToken: refreshToken})

	resp, err := authHandler.RefreshToken(ctx, req)
	if err != nil {
		ac.app.log.Errorf("Failed to refresh token: %v", err)
		return false
	}

	err = userEntity.Update().
		SetAccessToken(resp.Msg.AccessToken).
		SetAccessTokenExpiresAt(time.Now().Add(time.Duration(resp.Msg.AccessTokenExpiresIn) * time.Second)).
		SetRefreshToken(resp.Msg.RefreshToken).
		SetRefreshTokenExpiresAt(time.Now().Add(time.Duration(resp.Msg.RefreshTokenExpiresIn) * time.Second)).
		Exec(ctx)
	if err != nil {
		ac.app.log.Errorf("Failed to update local tokens for user %s: %v", userEntity.Email, err)
		return false
	}
	return true
}

// WaitForAuthentication handles authentication status streaming with a channel-based approach
func (ac *AuthClient) WaitForAuthentication(ctx context.Context, sessionID string, responseChan chan<- *v1.AuthStatusResponse) error {
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	authHandler := ac.getAuthHandler()
	req := connect.NewRequest(&v1.WaitForAuthRequest{SessionId: sessionID})

	// Open stream to external service
	externalStream, err := authHandler.RpcClient.WaitForAuthentication(ctx, req)
	if err != nil {
		return err
	}
	defer externalStream.Close()

	// Forward stream updates from external service to local client
	for externalStream.Receive() {
		resp := externalStream.Msg()

		// Send response to channel
		select {
		case responseChan <- resp:
		case <-ctx.Done():
			return ctx.Err()
		}

		// If authenticated, sync tokens locally and end stream
		if resp.Status == v1.AuthStatus_AUTHENTICATED {
			ac.syncAuthenticatedSession(ctx, sessionID, resp)
			return nil
		} else if resp.Status != v1.AuthStatus_PENDING {
			// Session expired or cancelled, update local session
			err = ac.app.db.AuthSession.Update().
				Where(authsession.IDEQ(sessionID)).
				SetStatus(authsession.Status(resp.Status.String())).
				Exec(ctx)
			if err != nil {
				ac.app.log.Errorf("Failed to update local session status: %v", err)
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

// syncAuthenticatedSession syncs tokens from external service to local db
// We only ever store one user, so get the first user or create one and update the email
func (ac *AuthClient) syncAuthenticatedSession(ctx context.Context, sessionID string, authResp *v1.AuthStatusResponse) {
	if authResp.User == nil {
		return
	}

	// Get the first user or create one (since we only store one user ever)
	var userEntity *ent.User
	existingUser, err := ac.app.db.User.Query().First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		ac.app.log.Errorf("Error querying for user: %v", err)
		return
	}

	if existingUser != nil {
		// Update existing user with new email and login time
		userEntity, err = existingUser.Update().
			SetEmail(authResp.User.Email).
			SetLastLoggedIn(time.Now()).
			Save(ctx)
		if err != nil {
			ac.app.log.Errorf("Error updating user: %v", err)
			return
		}
	} else {
		// Create new user
		userEntity, err = ac.app.db.User.Create().
			SetEmail(authResp.User.Email).
			SetLastLoggedIn(time.Now()).
			Save(ctx)
		if err != nil {
			ac.app.log.Errorf("Error creating user: %v", err)
			return
		}
	}

	// Store tokens locally on user
	updateQuery := ac.app.db.User.Update().Where(user.IDEQ(userEntity.ID))

	if authResp.RefreshToken != "" {
		updateQuery = updateQuery.SetRefreshToken(authResp.RefreshToken)
		if authResp.RefreshTokenExpiresIn > 0 {
			refreshExpiresAt := time.Now().Add(time.Duration(authResp.RefreshTokenExpiresIn) * time.Second)
			updateQuery = updateQuery.SetRefreshTokenExpiresAt(refreshExpiresAt)
		}
	}

	if authResp.AccessToken != "" {
		updateQuery = updateQuery.SetAccessToken(authResp.AccessToken)
		if authResp.AccessTokenExpiresIn > 0 {
			accessExpiresAt := time.Now().Add(time.Duration(authResp.AccessTokenExpiresIn) * time.Second)
			updateQuery = updateQuery.SetAccessTokenExpiresAt(accessExpiresAt)
		}
	}

	_, err = updateQuery.Save(ctx)
	if err != nil {
		ac.app.log.Errorf("Error saving tokens: %v", err)
	}

	// Update local session
	err = ac.app.db.AuthSession.Update().
		Where(authsession.IDEQ(sessionID)).
		SetStatus(authsession.StatusAUTHENTICATED).
		Exec(ctx)
	if err != nil {
		ac.app.log.Errorf("Error updating session status: %v", err)
	}
}

func (ac *AuthClient) RecoverAuthSessions(ctx context.Context) error {
	// Query for pending sessions that haven't expired
	pendingSessions, err := ac.app.db.AuthSession.Query().
		Where(authsession.StatusEQ(authsession.StatusPENDING)).
		Where(authsession.ExpiresAtGT(time.Now())).
		All(ctx)

	if err != nil {
		return err
	}

	// Start monitoring for each pending session
	for _, session := range pendingSessions {
		go ac.startAuthMonitoring(context.Background(), session.ID)
	}

	return nil
}

// validateAndRenewStoredTokens validates stored refresh tokens and automatically renews access tokens.
// Since we only store one user, this method gets the first user and attempts to refresh their tokens.
func (ac *AuthClient) validateAndRenewStoredTokens(ctx context.Context) error {
	// Delete expired refresh tokens
	err := ac.app.db.User.Update().
		Where(user.RefreshTokenExpiresAtLT(time.Now())).
		ClearRefreshToken().
		ClearAccessToken().
		ClearRefreshTokenExpiresAt().
		ClearAccessTokenExpiresAt().
		Exec(ctx)
	if err != nil {
		ac.app.log.Warnf("Failed to clear expired tokens: %v", err)
	}

	// Get the first user (since we only store one user ever)
	userEntity, err := ac.app.db.User.Query().
		Where(user.RefreshTokenNotNil()).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			ac.app.log.Info("No user with refresh token found")
			ac.eventEmitter.EmitEvent(ctx, types.EventNotAuthenticated.String())
			return nil
		}
		return fmt.Errorf("failed to query user with refresh token: %w", err)
	}

	// Check if access token is still valid
	if userEntity.AccessTokenExpiresAt != nil && userEntity.AccessTokenExpiresAt.After(time.Now().Add(5*time.Minute)) {
		// Access token is still valid for more than 5 minutes, no need to refresh
		ac.app.log.Debugf("Access token for user %s is still valid", userEntity.Email)
		ac.eventEmitter.EmitEvent(ctx, types.EventAuthenticated.String())
		return nil
	}

	// TODO: Move to keyring for better security
	refreshToken := *userEntity.RefreshToken

	// Attempt to refresh the token - RefreshToken returns success status
	success := ac.RefreshToken(ctx, refreshToken)

	if success {
		ac.app.log.Debugf("Successfully validated tokens for user %s", userEntity.Email)
		ac.eventEmitter.EmitEvent(ctx, types.EventAuthenticated.String())
	} else {
		ac.app.log.Info("Token refresh failed, no valid tokens found")
		ac.eventEmitter.EmitEvent(ctx, types.EventNotAuthenticated.String())
	}

	return nil
}

func (ac *AuthClient) startAuthMonitoring(ctx context.Context, sessionID string) {
	// Create a timeout context for the authentication monitoring (10 minutes total)
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	authHandler := ac.getAuthHandler()

	// Retry configuration
	const maxRetries = 20 // Max 20 retries over 10 minutes
	const retryInterval = 30 * time.Second
	retryCount := 0

	for retryCount <= maxRetries {
		select {
		case <-timeoutCtx.Done():
			// Overall timeout reached - emit not authenticated
			ac.eventEmitter.EmitEvent(context.Background(), types.EventNotAuthenticated.String())
			return
		default:
			// Continue with connection attempt
		}

		// Use WaitForAuthentication streaming approach
		req := connect.NewRequest(&v1.WaitForAuthRequest{SessionId: sessionID})

		stream, err := authHandler.RpcClient.WaitForAuthentication(timeoutCtx, req)
		if err != nil {
			retryCount++
			if retryCount > maxRetries {
				// Max retries exceeded - emit not authenticated event
				ac.eventEmitter.EmitEvent(context.Background(), types.EventNotAuthenticated.String())
				return
			}

			// Wait before retry
			select {
			case <-timeoutCtx.Done():
				ac.eventEmitter.EmitEvent(context.Background(), types.EventNotAuthenticated.String())
				return
			case <-time.After(retryInterval):
				continue
			}
		}

		// Stream established successfully - reset retry count
		retryCount = 0

		for stream.Receive() {
			authStatus := stream.Msg()

			switch authStatus.Status {
			case v1.AuthStatus_AUTHENTICATED:
				// Authentication successful - emit global authenticated event
				ac.eventEmitter.EmitEvent(context.Background(), types.EventAuthenticated.String())
				return
			case v1.AuthStatus_EXPIRED, v1.AuthStatus_CANCELLED:
				// Authentication failed - emit global not authenticated event
				ac.eventEmitter.EmitEvent(context.Background(), types.EventNotAuthenticated.String())
				return
			case v1.AuthStatus_PENDING:
				// Still pending - continue receiving
				continue
			default:
				// Unknown status - treat as not authenticated
				ac.eventEmitter.EmitEvent(context.Background(), types.EventNotAuthenticated.String())
				return
			}
		}

		// Stream ended - check for errors and potentially retry
		if err := stream.Err(); err != nil {
			retryCount++
			if retryCount > maxRetries {
				// Max retries exceeded - emit not authenticated event
				ac.eventEmitter.EmitEvent(context.Background(), types.EventNotAuthenticated.String())
				return
			}

			// Wait before retry
			select {
			case <-timeoutCtx.Done():
				ac.eventEmitter.EmitEvent(context.Background(), types.EventNotAuthenticated.String())
				return
			case <-time.After(retryInterval):
				continue
			}
		}

		// Stream ended without error (shouldn't happen normally)
		// Wait and retry
		retryCount++
		if retryCount > maxRetries {
			ac.eventEmitter.EmitEvent(context.Background(), types.EventNotAuthenticated.String())
			return
		}

		select {
		case <-timeoutCtx.Done():
			ac.eventEmitter.EmitEvent(context.Background(), types.EventNotAuthenticated.String())
			return
		case <-time.After(retryInterval):
			continue
		}
	}

	// All retries exhausted
	ac.eventEmitter.EmitEvent(context.Background(), types.EventNotAuthenticated.String())
}

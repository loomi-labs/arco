package auth

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/loomi-labs/arco/backend/api/v1/arcov1connect"
	"net/http"
	"time"

	v1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/authsession"
	"github.com/loomi-labs/arco/backend/ent/user"
	"go.uber.org/zap"
)

type AuthService struct {
	log       *zap.SugaredLogger
	db        *ent.Client
	state     *state.State
	rpcClient arcov1connect.AuthServiceClient
}

type AuthServicePublic interface {
	StartRegister(ctx context.Context, email string) error
	StartLogin(ctx context.Context, email string) error
	GetAuthState() state.AuthState
}

func NewAuthService(log *zap.SugaredLogger, db *ent.Client, state *state.State, cloudRPCURL string) *AuthService {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &AuthService{
		log:       log,
		db:        db,
		state:     state,
		rpcClient: arcov1connect.NewAuthServiceClient(httpClient, cloudRPCURL),
	}
}

func (as *AuthService) GetAuthState() state.AuthState {
	return as.state.GetAuthState()
}

func (as *AuthService) StartRegister(ctx context.Context, email string) error {
	req := connect.NewRequest(&v1.RegisterRequest{Email: email})

	// Make request to external auth service
	resp, err := as.rpcClient.Register(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to register with external auth service: %w", err)
	}

	// Store session locally for real-time updates
	sessionID := resp.Msg.SessionId
	expiresAt := resp.Msg.ExpiresAt.AsTime()
	_, err = as.db.AuthSession.Create().
		SetID(sessionID).
		SetStatus(authsession.StatusPENDING).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create local auth session: %w", err)
	}

	// Start monitoring authentication in the background
	go as.startAuthMonitoring(context.Background(), resp.Msg.SessionId)

	return nil
}

func (as *AuthService) StartLogin(ctx context.Context, email string) error {
	req := connect.NewRequest(&v1.LoginRequest{Email: email})

	// Make request to external auth service
	resp, err := as.rpcClient.Login(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}

	// Store session locally for real-time updates
	sessionID := resp.Msg.SessionId
	expiresAt := resp.Msg.ExpiresAt.AsTime()
	_, err = as.db.AuthSession.Create().
		SetID(sessionID).
		SetStatus(authsession.StatusPENDING).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create auth session: %w", err)
	}

	// Start monitoring authentication in the background
	go as.startAuthMonitoring(context.Background(), resp.Msg.SessionId)

	return nil
}

func (as *AuthService) refreshToken(ctx context.Context, refreshToken string) bool {
	// Update local token storage - get the first user since we only store one user
	userEntity, err := as.db.User.Query().First(ctx)
	if err != nil {
		as.log.Errorf("Failed to find user for token update: %v", err)
		return false
	}

	req := connect.NewRequest(&v1.RefreshTokenRequest{RefreshToken: refreshToken})

	resp, err := as.rpcClient.RefreshToken(ctx, req)
	if err != nil {
		as.log.Errorf("Failed to refresh token: %v", err)
		return false
	}

	err = as.updateUserTokens(ctx, userEntity, resp.Msg.AccessToken, resp.Msg.RefreshToken, resp.Msg.AccessTokenExpiresIn, resp.Msg.RefreshTokenExpiresIn)
	if err != nil {
		as.log.Errorf("Failed to update local tokens for user %s: %v", userEntity.Email, err)
		return false
	}
	return true
}

// syncAuthenticatedSession syncs tokens from external service to local db
// We only ever store one user, so get the first user or create one and update the email
func (as *AuthService) syncAuthenticatedSession(ctx context.Context, sessionID string, authResp *v1.AuthStatusResponse) {
	if authResp.User == nil {
		return
	}

	// Get the first user or create one (since we only store one user ever)
	var userEntity *ent.User
	existingUser, err := as.db.User.Query().First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		as.log.Errorf("Error querying for user: %v", err)
		return
	}

	if existingUser != nil {
		// Update existing user with new email and login time
		userEntity, err = existingUser.Update().
			SetEmail(authResp.User.Email).
			SetLastLoggedIn(time.Now()).
			Save(ctx)
		if err != nil {
			as.log.Errorf("Error updating user: %v", err)
			return
		}
	} else {
		// Create new user
		userEntity, err = as.db.User.Create().
			SetEmail(authResp.User.Email).
			SetLastLoggedIn(time.Now()).
			Save(ctx)
		if err != nil {
			as.log.Errorf("Error creating user: %v", err)
			return
		}
	}

	// Store tokens locally on user
	err = as.updateUserTokens(ctx, userEntity, authResp.AccessToken, authResp.RefreshToken, authResp.AccessTokenExpiresIn, authResp.RefreshTokenExpiresIn)
	if err != nil {
		as.log.Errorf("Error saving tokens: %v", err)
	}

	// Update local session
	err = as.db.AuthSession.Update().
		Where(authsession.IDEQ(sessionID)).
		SetStatus(authsession.StatusAUTHENTICATED).
		Exec(ctx)
	if err != nil {
		as.log.Errorf("Error updating session status: %v", err)
	}
}

func (as *AuthService) RecoverAuthSessions(ctx context.Context) error {
	// Query for pending sessions that haven't expired
	pendingSessions, err := as.db.AuthSession.Query().
		Where(authsession.StatusEQ(authsession.StatusPENDING)).
		Where(authsession.ExpiresAtGT(time.Now())).
		All(ctx)

	if err != nil {
		return err
	}

	// Start monitoring for each pending session
	for _, session := range pendingSessions {
		go as.startAuthMonitoring(context.Background(), session.ID)
	}

	return nil
}

// ValidateAndRenewStoredTokens validates stored refresh tokens and automatically renews access tokens.
// Since we only store one user, this method gets the first user and attempts to refresh their tokens.
func (as *AuthService) ValidateAndRenewStoredTokens(ctx context.Context) error {
	// Delete expired refresh tokens
	err := as.db.User.Update().
		Where(user.RefreshTokenExpiresAtLT(time.Now())).
		ClearRefreshToken().
		ClearAccessToken().
		ClearRefreshTokenExpiresAt().
		ClearAccessTokenExpiresAt().
		Exec(ctx)
	if err != nil {
		as.log.Warnf("Failed to clear expired tokens: %v", err)
	}

	// Get the first user (since we only store one user ever)
	userEntity, err := as.db.User.Query().
		Where(user.RefreshTokenNotNil()).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			as.log.Info("No user with refresh token found")
			as.state.SetNotAuthenticated(ctx)
			return nil
		}
		as.state.SetNotAuthenticated(ctx)
		return fmt.Errorf("failed to query user with refresh token: %w", err)
	}

	// Check if access token is still valid
	if userEntity.AccessTokenExpiresAt != nil && userEntity.AccessTokenExpiresAt.After(time.Now().Add(5*time.Minute)) {
		// Access token is still valid for more than 5 minutes, no need to refresh
		as.log.Debugf("Access token for user %s is still valid", userEntity.Email)
		as.state.SetAuthenticated(ctx)
		return nil
	}

	// TODO: Move to keyring for better security
	refreshToken := *userEntity.RefreshToken

	// Attempt to refresh the token - refreshToken returns success status
	success := as.refreshToken(ctx, refreshToken)

	if success {
		as.log.Debugf("Successfully validated tokens for user %s", userEntity.Email)
		as.state.SetAuthenticated(ctx)
	} else {
		as.log.Info("Token refresh failed, no valid tokens found")
		as.state.SetNotAuthenticated(ctx)
	}

	return nil
}

// updateUserTokens updates a user entity with the provided token information
func (as *AuthService) updateUserTokens(ctx context.Context, userEntity *ent.User, accessToken, refreshToken string, accessTokenExpiresIn, refreshTokenExpiresIn int64) error {
	updateQuery := userEntity.Update()

	// Set access token and expiration if provided
	if accessToken != "" {
		updateQuery = updateQuery.SetAccessToken(accessToken)
		if accessTokenExpiresIn > 0 {
			accessExpiresAt := time.Now().Add(time.Duration(accessTokenExpiresIn) * time.Second)
			updateQuery = updateQuery.SetAccessTokenExpiresAt(accessExpiresAt)
		}
	}

	// Set refresh token and expiration if provided
	if refreshToken != "" {
		updateQuery = updateQuery.SetRefreshToken(refreshToken)
		if refreshTokenExpiresIn > 0 {
			refreshExpiresAt := time.Now().Add(time.Duration(refreshTokenExpiresIn) * time.Second)
			updateQuery = updateQuery.SetRefreshTokenExpiresAt(refreshExpiresAt)
		}
	}

	return updateQuery.Exec(ctx)
}

func (as *AuthService) startAuthMonitoring(ctx context.Context, sessionID string) {
	// Create a timeout context for the authentication monitoring (10 minutes total)
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Retry configuration
	const maxRetries = 20 // Max 20 retries over 10 minutes
	const retryInterval = 30 * time.Second
	retryCount := 0

	for retryCount <= maxRetries {
		select {
		case <-timeoutCtx.Done():
			// Overall timeout reached - emit not authenticated
			as.state.SetNotAuthenticated(context.Background())
			return
		default:
			// Continue with connection attempt
		}

		// Use WaitForAuthentication streaming approach
		req := connect.NewRequest(&v1.WaitForAuthRequest{SessionId: sessionID})

		stream, err := as.rpcClient.WaitForAuthentication(timeoutCtx, req)
		if err != nil {
			retryCount++
			if retryCount > maxRetries {
				// Max retries exceeded - emit not authenticated event
				as.state.SetNotAuthenticated(context.Background())
				return
			}

			// Wait before retry
			select {
			case <-timeoutCtx.Done():
				as.state.SetNotAuthenticated(context.Background())
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
				// Authentication successful - store tokens and emit global authenticated event
				as.syncAuthenticatedSession(context.Background(), sessionID, authStatus)
				as.state.SetAuthenticated(context.Background())
				return
			case v1.AuthStatus_EXPIRED, v1.AuthStatus_CANCELLED:
				// Authentication failed - emit global not authenticated event
				as.state.SetNotAuthenticated(context.Background())
				return
			case v1.AuthStatus_PENDING:
				// Still pending - continue receiving
				continue
			default:
				// Unknown status - treat as not authenticated
				as.state.SetNotAuthenticated(context.Background())
				return
			}
		}

		// Stream ended - check for errors and potentially retry
		if err := stream.Err(); err != nil {
			retryCount++
			if retryCount > maxRetries {
				// Max retries exceeded - emit not authenticated event
				as.state.SetNotAuthenticated(context.Background())
				return
			}

			// Wait before retry
			select {
			case <-timeoutCtx.Done():
				as.state.SetNotAuthenticated(context.Background())
				return
			case <-time.After(retryInterval):
				continue
			}
		}

		// Stream ended without error (shouldn't happen normally)
		// Wait and retry
		retryCount++
		if retryCount > maxRetries {
			as.state.SetNotAuthenticated(context.Background())
			return
		}

		select {
		case <-timeoutCtx.Done():
			as.state.SetNotAuthenticated(context.Background())
			return
		case <-time.After(retryInterval):
			continue
		}
	}

	// All retries exhausted
	as.state.SetNotAuthenticated(context.Background())
}

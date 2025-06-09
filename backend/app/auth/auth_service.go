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

// AuthServiceInternal provides backend-only methods that should not be exposed to frontend
type AuthServiceInternal struct {
	*AuthService
}

func NewAuthService(log *zap.SugaredLogger, db *ent.Client, state *state.State, cloudRPCURL string) *AuthServiceInternal {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &AuthServiceInternal{
		AuthService: &AuthService{
			log:       log,
			db:        db,
			state:     state,
			rpcClient: arcov1connect.NewAuthServiceClient(httpClient, cloudRPCURL),
		},
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
	internal := &AuthServiceInternal{AuthService: as}
	go internal.startAuthMonitoring(context.Background(), resp.Msg.SessionId)

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
	internal := &AuthServiceInternal{AuthService: as}
	go internal.startAuthMonitoring(context.Background(), resp.Msg.SessionId)

	return nil
}

func (asi *AuthServiceInternal) refreshToken(ctx context.Context, refreshToken string) bool {
	// Update local token storage - get the first user since we only store one user
	userEntity, err := asi.db.User.Query().First(ctx)
	if err != nil {
		asi.log.Errorf("Failed to find user for token update: %v", err)
		return false
	}

	req := connect.NewRequest(&v1.RefreshTokenRequest{RefreshToken: refreshToken})

	resp, err := asi.rpcClient.RefreshToken(ctx, req)
	if err != nil {
		asi.log.Errorf("Failed to refresh token: %v", err)
		return false
	}

	err = asi.updateUserTokens(ctx, userEntity, resp.Msg.AccessToken, resp.Msg.RefreshToken, resp.Msg.AccessTokenExpiresIn, resp.Msg.RefreshTokenExpiresIn)
	if err != nil {
		asi.log.Errorf("Failed to update local tokens for user %s: %v", userEntity.Email, err)
		return false
	}
	return true
}

// syncAuthenticatedSession syncs tokens from external service to local db
// We only ever store one user, so get the first user or create one and update the email
func (asi *AuthServiceInternal) syncAuthenticatedSession(ctx context.Context, sessionID string, authResp *v1.AuthStatusResponse) {
	if authResp.User == nil {
		return
	}

	// Get the first user or create one (since we only store one user ever)
	var userEntity *ent.User
	existingUser, err := asi.db.User.Query().First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		asi.log.Errorf("Error querying for user: %v", err)
		return
	}

	if existingUser != nil {
		// Update existing user with new email and login time
		userEntity, err = existingUser.Update().
			SetEmail(authResp.User.Email).
			SetLastLoggedIn(time.Now()).
			Save(ctx)
		if err != nil {
			asi.log.Errorf("Error updating user: %v", err)
			return
		}
	} else {
		// Create new user
		userEntity, err = asi.db.User.Create().
			SetEmail(authResp.User.Email).
			SetLastLoggedIn(time.Now()).
			Save(ctx)
		if err != nil {
			asi.log.Errorf("Error creating user: %v", err)
			return
		}
	}

	// Store tokens locally on user
	err = asi.updateUserTokens(ctx, userEntity, authResp.AccessToken, authResp.RefreshToken, authResp.AccessTokenExpiresIn, authResp.RefreshTokenExpiresIn)
	if err != nil {
		asi.log.Errorf("Error saving tokens: %v", err)
	}

	// Update local session
	err = asi.db.AuthSession.Update().
		Where(authsession.IDEQ(sessionID)).
		SetStatus(authsession.StatusAUTHENTICATED).
		Exec(ctx)
	if err != nil {
		asi.log.Errorf("Error updating session status: %v", err)
	}
}

func (asi *AuthServiceInternal) RecoverAuthSessions(ctx context.Context) error {
	// Query for pending sessions that haven't expired
	pendingSessions, err := asi.db.AuthSession.Query().
		Where(authsession.StatusEQ(authsession.StatusPENDING)).
		Where(authsession.ExpiresAtGT(time.Now())).
		All(ctx)

	if err != nil {
		return err
	}

	// Start monitoring for each pending session
	for _, session := range pendingSessions {
		go asi.startAuthMonitoring(context.Background(), session.ID)
	}

	return nil
}

// ValidateAndRenewStoredTokens validates stored refresh tokens and automatically renews access tokens.
// Since we only store one user, this method gets the first user and attempts to refresh their tokens.
func (asi *AuthServiceInternal) ValidateAndRenewStoredTokens(ctx context.Context) error {
	// Delete expired refresh tokens
	err := asi.db.User.Update().
		Where(user.RefreshTokenExpiresAtLT(time.Now())).
		ClearRefreshToken().
		ClearAccessToken().
		ClearRefreshTokenExpiresAt().
		ClearAccessTokenExpiresAt().
		Exec(ctx)
	if err != nil {
		asi.log.Warnf("Failed to clear expired tokens: %v", err)
	}

	// Get the first user (since we only store one user ever)
	userEntity, err := asi.db.User.Query().
		Where(user.RefreshTokenNotNil()).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			asi.log.Info("No user with refresh token found")
			asi.state.SetNotAuthenticated(ctx)
			return nil
		}
		asi.state.SetNotAuthenticated(ctx)
		return fmt.Errorf("failed to query user with refresh token: %w", err)
	}

	// Check if access token is still valid
	if userEntity.AccessTokenExpiresAt != nil && userEntity.AccessTokenExpiresAt.After(time.Now().Add(5*time.Minute)) {
		// Access token is still valid for more than 5 minutes, no need to refresh
		asi.log.Debugf("Access token for user %s is still valid", userEntity.Email)
		asi.state.SetAuthenticated(ctx)
		return nil
	}

	// TODO: Move to keyring for better security
	refreshToken := *userEntity.RefreshToken

	// Attempt to refresh the token - refreshToken returns success status
	success := asi.refreshToken(ctx, refreshToken)

	if success {
		asi.log.Debugf("Successfully validated tokens for user %s", userEntity.Email)
		asi.state.SetAuthenticated(ctx)
	} else {
		asi.log.Info("Token refresh failed, no valid tokens found")
		asi.state.SetNotAuthenticated(ctx)
	}

	return nil
}

// updateUserTokens updates a user entity with the provided token information
func (asi *AuthServiceInternal) updateUserTokens(ctx context.Context, userEntity *ent.User, accessToken, refreshToken string, accessTokenExpiresIn, refreshTokenExpiresIn int64) error {
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

func (asi *AuthServiceInternal) startAuthMonitoring(ctx context.Context, sessionID string) {
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
			asi.state.SetNotAuthenticated(context.Background())
			return
		default:
			// Continue with connection attempt
		}

		// Use WaitForAuthentication streaming approach
		req := connect.NewRequest(&v1.WaitForAuthRequest{SessionId: sessionID})

		stream, err := asi.rpcClient.WaitForAuthentication(timeoutCtx, req)
		if err != nil {
			retryCount++
			if retryCount > maxRetries {
				// Max retries exceeded - emit not authenticated event
				asi.state.SetNotAuthenticated(context.Background())
				return
			}

			// Wait before retry
			select {
			case <-timeoutCtx.Done():
				asi.state.SetNotAuthenticated(context.Background())
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
				asi.syncAuthenticatedSession(context.Background(), sessionID, authStatus)
				asi.state.SetAuthenticated(context.Background())
				return
			case v1.AuthStatus_EXPIRED, v1.AuthStatus_CANCELLED:
				// Authentication failed - emit global not authenticated event
				asi.state.SetNotAuthenticated(context.Background())
				return
			case v1.AuthStatus_PENDING:
				// Still pending - continue receiving
				continue
			default:
				// Unknown status - treat as not authenticated
				asi.state.SetNotAuthenticated(context.Background())
				return
			}
		}

		// Stream ended - check for errors and potentially retry
		if err := stream.Err(); err != nil {
			retryCount++
			if retryCount > maxRetries {
				// Max retries exceeded - emit not authenticated event
				asi.state.SetNotAuthenticated(context.Background())
				return
			}

			// Wait before retry
			select {
			case <-timeoutCtx.Done():
				asi.state.SetNotAuthenticated(context.Background())
				return
			case <-time.After(retryInterval):
				continue
			}
		}

		// Stream ended without error (shouldn't happen normally)
		// Wait and retry
		retryCount++
		if retryCount > maxRetries {
			asi.state.SetNotAuthenticated(context.Background())
			return
		}

		select {
		case <-timeoutCtx.Done():
			asi.state.SetNotAuthenticated(context.Background())
			return
		case <-time.After(retryInterval):
			continue
		}
	}

	// All retries exhausted
	asi.state.SetNotAuthenticated(context.Background())
}

func (as *AuthService) SetDb(db *ent.Client) {
	as.db = db
}

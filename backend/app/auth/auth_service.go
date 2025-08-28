package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/loomi-labs/arco/backend/api/v1/arcov1connect"

	v1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/authsession"
	"github.com/loomi-labs/arco/backend/ent/user"
	"go.uber.org/zap"
)

type AuthStatus string

const (
	AuthStatusSuccess         AuthStatus = "success"
	AuthStatusRateLimitError  AuthStatus = "rateLimitError"
	AuthStatusConnectionError AuthStatus = "connectionError"
	AuthStatusError           AuthStatus = "error"
)

type Service struct {
	log       *zap.SugaredLogger
	db        *ent.Client
	state     *state.State
	rpcClient arcov1connect.AuthServiceClient
}

// ServiceInternal provides backend-only methods that should not be exposed to frontend
type ServiceInternal struct {
	*Service
}

func NewService(log *zap.SugaredLogger, state *state.State) *ServiceInternal {
	return &ServiceInternal{
		Service: &Service{
			log:   log,
			state: state,
		},
	}
}

func (asi *ServiceInternal) Init(db *ent.Client, rpcClient arcov1connect.AuthServiceClient) {
	asi.db = db
	asi.rpcClient = rpcClient
}

// mustHaveDB panics if db is nil. This is a programming error guard.
func (as *Service) mustHaveDB() {
	if as.db == nil {
		panic("AuthService: database client is nil")
	}
}

func (as *Service) GetAuthState() state.AuthState {
	return as.state.GetAuthState()
}

func (as *Service) StartRegister(ctx context.Context, email string) (AuthStatus, error) {
	as.mustHaveDB()

	req := connect.NewRequest(&v1.RegisterRequest{Email: email})

	// Make request to external auth service
	resp, err := as.rpcClient.Register(ctx, req)
	if err != nil {
		// Map gRPC error codes to user-friendly errors
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			switch connectErr.Code() {
			case connect.CodeResourceExhausted:
				as.log.Errorf("Rate limit exceeded for registration: %v", err)
				return AuthStatusRateLimitError, nil
			case connect.CodeUnavailable, connect.CodeDeadlineExceeded, connect.CodeAborted:
				as.log.Errorf("Connection error during registration: %v", err)
				return AuthStatusConnectionError, nil
			case connect.CodeNotFound:
				as.log.Errorf("Service unavailable during registration: %v", err)
				return AuthStatusError, nil
			default:
				as.log.Errorf("Registration failed: %v", err)
				return AuthStatusError, fmt.Errorf("failed to register")
			}
		}
		as.log.Errorf("Failed to register with external auth service: %v", err)
		return AuthStatusError, fmt.Errorf("failed to register")
	}

	// Store session locally for real-time updates
	sessionID := resp.Msg.SessionId
	expiresAt := time.Now().Add(time.Duration(resp.Msg.ExpiresIn) * time.Second)
	session, err := as.db.AuthSession.Create().
		SetSessionID(sessionID).
		SetStatus(authsession.StatusPENDING).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		as.log.Errorf("Failed to create local auth session: %v", err)
		return AuthStatusError, fmt.Errorf("failed to register")
	}

	// Start monitoring authentication in the background
	go as.startAuthMonitoring(session)

	return AuthStatusSuccess, nil
}

func (as *Service) StartLogin(ctx context.Context, email string) (AuthStatus, error) {
	as.mustHaveDB()

	req := connect.NewRequest(&v1.LoginRequest{Email: email})

	// Make request to external auth service
	resp, err := as.rpcClient.Login(ctx, req)
	if err != nil {
		// Map gRPC error codes to user-friendly errors
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			switch connectErr.Code() {
			case connect.CodeResourceExhausted:
				as.log.Errorf("Rate limit exceeded for login: %v", err)
				return AuthStatusRateLimitError, nil
			case connect.CodeUnavailable, connect.CodeDeadlineExceeded, connect.CodeAborted:
				as.log.Errorf("Connection error during login: %v", err)
				return AuthStatusConnectionError, nil
			case connect.CodeNotFound:
				as.log.Errorf("No account found for email during login: %v", err)
				return AuthStatusError, nil
			default:
				as.log.Errorf("Login failed: %v", err)
				return AuthStatusError, fmt.Errorf("login failed")
			}
		}
		as.log.Errorf("Failed to login: %v", err)
		return AuthStatusError, fmt.Errorf("failed to login")
	}

	// Store session locally for real-time updates
	sessionID := resp.Msg.SessionId
	expiresAt := time.Now().Add(time.Duration(resp.Msg.ExpiresIn) * time.Second)
	session, err := as.db.AuthSession.Create().
		SetSessionID(sessionID).
		SetStatus(authsession.StatusPENDING).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		as.log.Errorf("Failed to create auth session: %v", err)
		return AuthStatusError, fmt.Errorf("failed to login")
	}

	// Start monitoring authentication in the background
	go as.startAuthMonitoring(session)

	return AuthStatusSuccess, nil
}

func (as *Service) Logout(ctx context.Context) error {
	as.mustHaveDB()

	// Get the user's refresh token for logout request
	userEntity, err := as.db.User.Query().
		Where(user.RefreshTokenNotNil()).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			// No user with refresh token found, already logged out
			as.state.SetNotAuthenticated(ctx)
			return nil
		}
		return fmt.Errorf("failed to query user: %w", err)
	}

	// Clear tokens locally
	err = userEntity.Update().
		ClearRefreshToken().
		ClearAccessToken().
		ClearRefreshTokenExpiresAt().
		ClearAccessTokenExpiresAt().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to clear tokens: %w", err)
	}

	// Set authentication state
	as.state.SetNotAuthenticated(ctx)

	return nil
}

func (asi *ServiceInternal) refreshToken(ctx context.Context, userEntity *ent.User) {
	asi.mustHaveDB()

	req := connect.NewRequest(&v1.RefreshTokenRequest{RefreshToken: *userEntity.RefreshToken})

	resp, err := asi.rpcClient.RefreshToken(ctx, req)
	if err != nil {
		asi.log.Errorf("Failed to refresh token: %v", err)
		return
	}

	err = asi.updateUserTokens(ctx, userEntity, resp.Msg.AccessToken, resp.Msg.RefreshToken, resp.Msg.AccessTokenExpiresIn, resp.Msg.RefreshTokenExpiresIn)
	if err != nil {
		asi.log.Errorf("Failed to update local tokens for user %s: %v", userEntity.Email, err)
		return
	}
	return
}

// syncAuthenticatedSession syncs tokens from external service to local db
// We only ever store one user, so get the first user or create one and update the email
func (asi *ServiceInternal) syncAuthenticatedSession(ctx context.Context, sessionID string, authResp *v1.WaitForAuthenticationResponse) {
	asi.mustHaveDB()

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
		Where(authsession.SessionIDEQ(sessionID)).
		SetStatus(authsession.StatusAUTHENTICATED).
		Exec(ctx)
	if err != nil {
		asi.log.Errorf("Error updating session status: %v", err)
	}
}

func (asi *ServiceInternal) RecoverAuthSessions(ctx context.Context) error {
	asi.mustHaveDB()

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
		go asi.startAuthMonitoring(session)
	}

	return nil
}

// ValidateAndRenewStoredTokens validates stored refresh tokens and automatically renews access tokens.
// Since we only store one user, this method gets the first user and attempts to refresh their tokens.
func (asi *ServiceInternal) ValidateAndRenewStoredTokens(ctx context.Context) error {
	asi.mustHaveDB()

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

	// Attempt to refresh the token
	asi.refreshToken(ctx, userEntity)

	return nil
}

// updateUserTokens updates a user entity with the provided token information
func (asi *ServiceInternal) updateUserTokens(ctx context.Context, userEntity *ent.User, accessToken, refreshToken string, accessTokenExpiresIn, refreshTokenExpiresIn int64) error {
	asi.log.Debugf("Updating tokens for user %s (access expires in %d seconds, refresh expires in %d seconds)", userEntity.Email, accessTokenExpiresIn, refreshTokenExpiresIn)

	if accessToken == "" || accessTokenExpiresIn <= 0 || refreshToken == "" || refreshTokenExpiresIn <= 0 {
		asi.log.Errorf("Got invalid value for JWT's for user %s", userEntity.Email)
		return fmt.Errorf("invalid value for JWT's for user")
	}

	accessExpiresAt := time.Now().Add(time.Duration(float64(accessTokenExpiresIn)*0.9) * time.Second)   // Apply 10% buffer to token expiration
	refreshExpiresAt := time.Now().Add(time.Duration(float64(refreshTokenExpiresIn)*0.9) * time.Second) // Apply 10% buffer to token expiration
	err := userEntity.Update().
		SetAccessToken(accessToken).
		SetAccessTokenExpiresAt(accessExpiresAt).
		SetRefreshToken(refreshToken).
		SetRefreshTokenExpiresAt(refreshExpiresAt).
		Exec(ctx)
	if err != nil {
		asi.state.SetNotAuthenticated(ctx)
	} else {
		asi.state.SetAuthenticated(ctx)
	}
	return err
}

func (as *Service) startAuthMonitoring(session *ent.AuthSession) {
	// Create a timeout context based on session expiration
	timeout := time.Until(session.ExpiresAt)
	if timeout <= 0 {
		// Session already expired
		as.log.Debugf("Session %s: already expired", session.SessionID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Retry configuration
	const maxRetries = 120 // Max 120 retries over 10 minutes
	const retryInterval = 5 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Use WaitForAuthentication streaming approach
		req := connect.NewRequest(&v1.WaitForAuthenticationRequest{SessionId: session.SessionID})

		as.log.Debugf("Session %s: attempting auth stream (attempt %d/%d)", session.SessionID, attempt+1, maxRetries)
		stream, err := as.rpcClient.WaitForAuthentication(ctx, req)
		if err != nil {
			as.log.Debugf("Session %s: stream connection failed: %v", session.SessionID, err)
			if attempt == maxRetries-1 {
				as.log.Debugf("Session %s: max retries reached, stopping monitoring", session.SessionID)
				return
			}

			// Wait before retry
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryInterval):
				continue
			}
		}

		// Stream established successfully
		as.log.Debugf("Session %s: auth stream established", session.SessionID)

		for stream.Receive() {
			authStatus := stream.Msg()

			switch authStatus.Status {
			case v1.AuthStatus_AUTH_STATUS_AUTHENTICATED:
				// Authentication successful
				as.log.Debugf("Session %s: authentication successful", session.SessionID)
				internal := &ServiceInternal{Service: as}
				internal.syncAuthenticatedSession(ctx, session.SessionID, authStatus)
				as.state.SetAuthenticated(ctx)
				return
			case v1.AuthStatus_AUTH_STATUS_EXPIRED, v1.AuthStatus_AUTH_STATUS_CANCELLED:
				// Authentication failed
				as.log.Debugf("Session %s: authentication failed with status %v", session.SessionID, authStatus.Status)
				return
			case v1.AuthStatus_AUTH_STATUS_PENDING:
				// Still pending - continue waiting
				as.log.Debugf("Session %s: pending authentication", session.SessionID)
				continue
			case v1.AuthStatus_AUTH_STATUS_UNSPECIFIED:
				// Unknown status
				as.log.Debugf("Session %s: unknown auth status %v", session.SessionID, authStatus.Status)
				return
			}
		}

		// Stream ended - check for errors and retry if not max attempts
		if err := stream.Err(); err != nil {
			as.log.Debugf("Session %s: stream error: %v", session.SessionID, err)
			if attempt == maxRetries-1 {
				as.log.Debugf("Session %s: max retries reached after error", session.SessionID)
				return
			}

			// Wait before retry
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryInterval):
				continue
			}
		}

		// Stream ended without error - retry
		as.log.Debugf("Session %s: stream ended, retrying", session.SessionID)
	}

	as.log.Debugf("Session %s: monitoring completed", session.SessionID)
}

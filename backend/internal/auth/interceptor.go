package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/loomi-labs/arco/backend/app/auth"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/internal/keyring"
	"go.uber.org/zap"
)

// JWTAuthInterceptor provides JWT authentication for Connect RPC clients
type JWTAuthInterceptor struct {
	log            *zap.SugaredLogger
	authServiceRPC *auth.ServiceInternal
	db             *ent.Client
	state          *state.State
	keyring        *keyring.Service
}

// NewJWTAuthInterceptor creates a new JWT authentication interceptor
func NewJWTAuthInterceptor(log *zap.SugaredLogger, authServiceRPC *auth.ServiceInternal, db *ent.Client, state *state.State, keyring *keyring.Service) *JWTAuthInterceptor {
	return &JWTAuthInterceptor{
		log:            log,
		authServiceRPC: authServiceRPC,
		db:             db,
		state:          state,
		keyring:        keyring,
	}
}

// mustHaveDB panics if db is nil. This is a programming error guard.
func (j *JWTAuthInterceptor) mustHaveDB() {
	if j.db == nil {
		panic("JWTAuthInterceptor: database client is nil")
	}
}

// UnaryInterceptor returns a Connect RPC interceptor that adds JWT authentication
// and handles unauthenticated responses by clearing stored tokens
func (j *JWTAuthInterceptor) UnaryInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Only add auth headers to client requests (outgoing calls)
			if !req.Spec().IsClient {
				return next(ctx, req)
			}

			// Get the current user's access token
			token, err := j.getCurrentAccessToken(ctx)
			if err != nil {
				j.log.Debugf("No valid access token available: %v", err)
				// Continue without token - let the service handle the authentication error
				return next(ctx, req)
			}

			// Add Authorization header with Bearer token
			req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))

			// Execute the request
			resp, err := next(ctx, req)

			// Check for unauthenticated errors and clear tokens if needed
			if err != nil {
				var connectErr *connect.Error
				if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeUnauthenticated {
					j.log.Debug("Received unauthenticated error, clearing stored tokens")
					if clearErr := j.clearTokens(ctx); clearErr != nil {
						j.log.Errorf("Failed to clear tokens after unauthenticated error: %v", clearErr)
					}
				}
			}

			return resp, err
		}
	}
}

// getCurrentAccessToken retrieves a valid access token for the current user
func (j *JWTAuthInterceptor) getCurrentAccessToken(ctx context.Context) (string, error) {
	j.mustHaveDB()

	// Try to get access token from keyring
	accessToken, err := j.keyring.GetAccessToken()
	if err != nil {
		return "", fmt.Errorf("no access token in keyring: %w", err)
	}

	// Get the first user to check token expiry (since we only store one user)
	user, err := j.db.User.Query().First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", fmt.Errorf("no authenticated user found")
		}
		return "", fmt.Errorf("failed to query user: %w", err)
	}

	// Check if token is expired and refresh if needed
	if user.AccessTokenExpiresAt != nil && user.AccessTokenExpiresAt.Before(time.Now().Add(30*time.Second)) {
		j.log.Debug("Access token expired, attempting refresh")

		// Validate and refresh tokens - this will update keyring
		err = j.authServiceRPC.ValidateAndRenewStoredTokens(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to refresh token: %w", err)
		}

		// Get the refreshed token from keyring
		accessToken, err = j.keyring.GetAccessToken()
		if err != nil {
			return "", fmt.Errorf("no access token after refresh: %w", err)
		}
	}

	return accessToken, nil
}

// clearTokens clears all stored authentication tokens when receiving unauthenticated errors
func (j *JWTAuthInterceptor) clearTokens(ctx context.Context) error {
	j.mustHaveDB()

	// Clear tokens from keyring
	if err := j.keyring.DeleteTokens(); err != nil {
		j.log.Warnf("Failed to delete tokens from keyring: %v", err)
	}

	// Clear token expiry times from database
	err := j.db.User.Update().
		ClearRefreshTokenExpiresAt().
		ClearAccessTokenExpiresAt().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to clear token expiry times: %w", err)
	}
	j.state.SetNotAuthenticated(ctx)

	j.log.Info("Cleared authentication tokens due to unauthenticated response")
	return nil
}

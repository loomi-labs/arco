package auth

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/loomi-labs/arco/backend/app/auth"
	"github.com/loomi-labs/arco/backend/ent"
	"go.uber.org/zap"
)

// JWTAuthInterceptor provides JWT authentication for Connect RPC clients
type JWTAuthInterceptor struct {
	log            *zap.SugaredLogger
	authServiceRPC *auth.ServiceInternal
	db             *ent.Client
}

// NewJWTAuthInterceptor creates a new JWT authentication interceptor
func NewJWTAuthInterceptor(log *zap.SugaredLogger, authServiceRPC *auth.ServiceInternal, db *ent.Client) *JWTAuthInterceptor {
	return &JWTAuthInterceptor{
		log:            log,
		authServiceRPC: authServiceRPC,
		db:             db,
	}
}

// UnaryInterceptor returns a Connect RPC interceptor that adds JWT authentication
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

			return next(ctx, req)
		}
	}
}

// getCurrentAccessToken retrieves a valid access token for the current user
func (j *JWTAuthInterceptor) getCurrentAccessToken(ctx context.Context) (string, error) {
	if j.db == nil {
		return "", fmt.Errorf("database client not available")
	}

	// Get the first user (since we only store one user)
	user, err := j.db.User.Query().First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", fmt.Errorf("no authenticated user found")
		}
		return "", fmt.Errorf("failed to query user: %w", err)
	}

	// Check if user has an access token
	if user.AccessToken == nil || *user.AccessToken == "" {
		return "", fmt.Errorf("user has no access token")
	}

	// Check if token is expired and refresh if needed
	if user.AccessTokenExpiresAt != nil && user.AccessTokenExpiresAt.Before(time.Now()) {
		j.log.Debug("Access token expired, attempting refresh")

		// Validate and refresh tokens - this will update the user in the database
		err = j.authServiceRPC.ValidateAndRenewStoredTokens(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to refresh token: %w", err)
		}

		// Re-fetch the user to get the updated token
		user, err = j.db.User.Query().First(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get refreshed user: %w", err)
		}

		if user.AccessToken == nil || *user.AccessToken == "" {
			return "", fmt.Errorf("no access token after refresh")
		}
	}

	return *user.AccessToken, nil
}

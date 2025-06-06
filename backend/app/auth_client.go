package app

import (
	"connectrpc.com/connect"
	"context"
	"time"

	v1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/app/auth"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/ent/authsession"
)

type AuthClient struct {
	app          *App
	eventEmitter types.EventEmitter
}

var (
	// Shared instances to maintain session state
	globalJWTService  *auth.JWTService
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
		globalJWTService = auth.NewJWTService("your-jwt-secret-key") // TODO: Move to config
		globalAuthHandler = auth.NewAuthServiceHandler(ac.app.db, globalJWTService, ac.app.config.CloudRPCURL)
	}
	return globalAuthHandler
}

func (ac *AuthClient) StartRegister(ctx context.Context, email string) (*v1.RegisterResponse, error) {
	authHandler := ac.getAuthHandler()

	req := connect.NewRequest(&v1.RegisterRequest{Email: email})

	resp, err := authHandler.Register(ctx, req)
	if err != nil {
		return nil, err
	}

	// Start monitoring authentication in the background
	go ac.startAuthMonitoring(context.Background(), resp.Msg.SessionId)

	return resp.Msg, nil
}

func (ac *AuthClient) StartLogin(ctx context.Context, email string) (*v1.LoginResponse, error) {
	authHandler := ac.getAuthHandler()

	req := connect.NewRequest(&v1.LoginRequest{Email: email})

	resp, err := authHandler.Login(ctx, req)
	if err != nil {
		return nil, err
	}

	// Start monitoring authentication in the background
	go ac.startAuthMonitoring(context.Background(), resp.Msg.SessionId)

	return resp.Msg, nil
}

func (ac *AuthClient) RefreshToken(ctx context.Context, refreshToken string) (*v1.RefreshTokenResponse, error) {
	authHandler := ac.getAuthHandler()

	req := connect.NewRequest(&v1.RefreshTokenRequest{RefreshToken: refreshToken})

	resp, err := authHandler.RefreshToken(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Msg, nil
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

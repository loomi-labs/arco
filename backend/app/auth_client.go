package app

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	v1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/app/auth"
)

type AuthClient App

var (
	// Shared instances to maintain session state
	globalJWTService  *auth.JWTService
	globalAuthHandler *auth.AuthServiceHandler
)

func (a *App) AuthClient() *AuthClient {
	return (*AuthClient)(a)
}

func (ac *AuthClient) getAuthHandler() *auth.AuthServiceHandler {
	if globalAuthHandler == nil {
		if ac.db == nil {
			return nil
		}
		globalJWTService = auth.NewJWTService("your-jwt-secret-key") // TODO: Move to config
		globalAuthHandler = auth.NewAuthServiceHandler(ac.db, globalJWTService, ac.config.CloudRPCURL)
	}
	return globalAuthHandler
}

func (ac *AuthClient) StartRegister(ctx context.Context, email string) (*v1.RegisterResponse, error) {
	if ac.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	authHandler := ac.getAuthHandler()
	if authHandler == nil {
		return nil, fmt.Errorf("failed to initialize auth handler")
	}

	req := connect.NewRequest(&v1.RegisterRequest{Email: email})

	resp, err := authHandler.Register(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Msg, nil
}

func (ac *AuthClient) StartLogin(ctx context.Context, email string) (*v1.LoginResponse, error) {
	if ac.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	authHandler := ac.getAuthHandler()
	if authHandler == nil {
		return nil, fmt.Errorf("failed to initialize auth handler")
	}

	req := connect.NewRequest(&v1.LoginRequest{Email: email})

	resp, err := authHandler.Login(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Msg, nil
}

func (ac *AuthClient) CheckAuthStatus(ctx context.Context, sessionID string) (*v1.CheckAuthStatusResponse, error) {
	if ac.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	authHandler := ac.getAuthHandler()
	if authHandler == nil {
		return nil, fmt.Errorf("failed to initialize auth handler")
	}

	req := connect.NewRequest(&v1.CheckAuthStatusRequest{SessionId: sessionID})

	resp, err := authHandler.CheckAuthStatus(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Msg, nil
}

func (ac *AuthClient) RefreshToken(ctx context.Context, refreshToken string) (*v1.RefreshTokenResponse, error) {
	if ac.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	authHandler := ac.getAuthHandler()
	if authHandler == nil {
		return nil, fmt.Errorf("failed to initialize auth handler")
	}

	req := connect.NewRequest(&v1.RefreshTokenRequest{RefreshToken: refreshToken})

	resp, err := authHandler.RefreshToken(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Msg, nil
}

func (ac *AuthClient) CompleteAuthentication(ctx context.Context, sessionID string) error {
	if ac.db == nil {
		return fmt.Errorf("database not initialized")
	}

	authHandler := ac.getAuthHandler()
	if authHandler == nil {
		return fmt.Errorf("failed to initialize auth handler")
	}

	return authHandler.CompleteAuthentication(ctx, sessionID)
}

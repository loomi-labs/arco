package auth

import (
	"context"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
	EmailKey  contextKey = "email"
)

type AuthMiddleware struct {
	jwtService *JWTService
}

func NewAuthMiddleware(jwtService *JWTService) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
	}
}

func (m *AuthMiddleware) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			http.Error(w, "Bearer token required", http.StatusUnauthorized)
			return
		}

		claims, err := m.jwtService.ValidateAccessToken(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			http.Error(w, "Invalid user ID in token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		ctx = context.WithValue(ctx, EmailKey, claims.Email)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func (m *AuthMiddleware) ConnectInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			authHeader := req.Header().Get("Authorization")
			if authHeader == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, nil)
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == authHeader {
				return nil, connect.NewError(connect.CodeUnauthenticated, nil)
			}

			claims, err := m.jwtService.ValidateAccessToken(token)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}

			userID, err := uuid.Parse(claims.UserID)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}

			ctx = context.WithValue(ctx, UserIDKey, userID)
			ctx = context.WithValue(ctx, EmailKey, claims.Email)

			return next(ctx, req)
		}
	}
}

func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
	return userID, ok
}

func GetEmail(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(EmailKey).(string)
	return email, ok
}

func RequireAuth(ctx context.Context) (uuid.UUID, string, error) {
	userID, ok := GetUserID(ctx)
	if !ok {
		return uuid.Nil, "", connect.NewError(connect.CodeUnauthenticated, nil)
	}

	email, ok := GetEmail(ctx)
	if !ok {
		return uuid.Nil, "", connect.NewError(connect.CodeUnauthenticated, nil)
	}

	return userID, email, nil
}

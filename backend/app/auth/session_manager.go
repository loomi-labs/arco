package auth

import (
	"context"
	"sync"
	"time"

	v1 "github.com/loomi-labs/arco/backend/api/v1"
)

type SessionStatus string

const (
	SessionPending       SessionStatus = "PENDING"
	SessionAuthenticated SessionStatus = "AUTHENTICATED"
	SessionExpired       SessionStatus = "EXPIRED"
	SessionCancelled     SessionStatus = "CANCELLED"
)

type SessionData struct {
	ID        string
	UserEmail string
	Status    SessionStatus
	ExpiresAt time.Time
	UserID    string
	Tokens    *TokenPair
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*SessionData
	streams  map[string][]chan *v1.AuthStatusResponse
}

func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*SessionData),
		streams:  make(map[string][]chan *v1.AuthStatusResponse),
	}

	go sm.cleanupExpiredSessions()
	return sm
}

func (sm *SessionManager) CreateSession(sessionID, userEmail string) *SessionData {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := &SessionData{
		ID:        sessionID,
		UserEmail: userEmail,
		Status:    SessionPending,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	sm.sessions[sessionID] = session
	return session
}

func (sm *SessionManager) GetSession(sessionID string) (*SessionData, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, false
	}

	if time.Now().After(session.ExpiresAt) && session.Status == SessionPending {
		session.Status = SessionExpired
	}

	return session, true
}

func (sm *SessionManager) UpdateSessionStatus(sessionID string, status SessionStatus, userID string, tokens *TokenPair) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return false
	}

	session.Status = status
	if userID != "" {
		session.UserID = userID
	}
	if tokens != nil {
		session.Tokens = tokens
	}

	sm.broadcastToStreams(sessionID, session)
	return true
}

func (sm *SessionManager) AddStream(sessionID string, stream chan *v1.AuthStatusResponse) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.streams[sessionID] == nil {
		sm.streams[sessionID] = make([]chan *v1.AuthStatusResponse, 0)
	}
	sm.streams[sessionID] = append(sm.streams[sessionID], stream)
}

func (sm *SessionManager) RemoveStream(sessionID string, stream chan *v1.AuthStatusResponse) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	streams := sm.streams[sessionID]
	for i, s := range streams {
		if s == stream {
			sm.streams[sessionID] = append(streams[:i], streams[i+1:]...)
			break
		}
	}

	if len(sm.streams[sessionID]) == 0 {
		delete(sm.streams, sessionID)
	}
}

func (sm *SessionManager) broadcastToStreams(sessionID string, session *SessionData) {
	streams := sm.streams[sessionID]
	if len(streams) == 0 {
		return
	}

	response := sm.sessionToResponse(session)

	for _, stream := range streams {
		select {
		case stream <- response:
		default:
		}
	}
}

func (sm *SessionManager) sessionToResponse(session *SessionData) *v1.AuthStatusResponse {
	response := &v1.AuthStatusResponse{}

	switch session.Status {
	case SessionPending:
		response.Status = v1.AuthStatus_PENDING
	case SessionAuthenticated:
		response.Status = v1.AuthStatus_AUTHENTICATED
		if session.Tokens != nil {
			response.AccessToken = session.Tokens.AccessToken
			response.RefreshToken = session.Tokens.RefreshToken
			response.ExpiresIn = session.Tokens.ExpiresIn
		}
		if session.UserID != "" {
			response.User = &v1.User{
				Id:    session.UserID,
				Email: session.UserEmail,
			}
		}
	case SessionExpired:
		response.Status = v1.AuthStatus_EXPIRED
	case SessionCancelled:
		response.Status = v1.AuthStatus_CANCELLED
	}

	return response
}

func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		for sessionID, session := range sm.sessions {
			if now.After(session.ExpiresAt) && session.Status == SessionPending {
				session.Status = SessionExpired
				sm.broadcastToStreams(sessionID, session)

				time.AfterFunc(5*time.Minute, func() {
					sm.mu.Lock()
					defer sm.mu.Unlock()
					delete(sm.sessions, sessionID)
					delete(sm.streams, sessionID)
				})
			}
		}
		sm.mu.Unlock()
	}
}

func (sm *SessionManager) StreamSessionUpdates(ctx context.Context, sessionID string) chan *v1.AuthStatusResponse {
	stream := make(chan *v1.AuthStatusResponse, 10)
	sm.AddStream(sessionID, stream)

	go func() {
		defer func() {
			sm.RemoveStream(sessionID, stream)
			close(stream)
		}()

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		timeout := time.NewTimer(10 * time.Minute)
		defer timeout.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-timeout.C:
				return
			case <-ticker.C:
				if session, exists := sm.GetSession(sessionID); exists {
					response := sm.sessionToResponse(session)
					select {
					case stream <- response:
						if session.Status != SessionPending {
							return
						}
					case <-ctx.Done():
						return
					}
				} else {
					return
				}
			}
		}
	}()

	return stream
}

package keyring

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/99designs/keyring"
	"github.com/godbus/dbus/v5"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/platform"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	ServiceName = "arco-backup"

	// Key prefix for all arco secrets in the default keyring collection
	keyPrefix = "arco-backup:"

	// Key format for repository passwords
	repoPasswordKeyFmt = keyPrefix + "repo:%d:password"

	// Keys for auth tokens
	accessTokenKey  = keyPrefix + "user:access_token"
	refreshTokenKey = keyPrefix + "user:refresh_token"
)

// Service provides secure credential storage using the system keyring
type Service struct {
	log  *zap.SugaredLogger
	ring keyring.Keyring

	// In-memory token cache
	mu                 sync.RWMutex
	cachedAccessToken  string
	cachedRefreshToken string
}

// NewService creates a new keyring service with platform-appropriate backends
func NewService(log *zap.SugaredLogger, config *types.Config) (*Service, error) {
	// Ensure keyring directory exists with restrictive permissions
	if err := os.MkdirAll(config.KeyringDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create keyring directory: %w", err)
	}

	// File backend uses a static key - credentials are effectively stored in cleartext.
	// Real security comes from system keyrings (GNOME Keyring, macOS Keychain, Windows Credential Manager).
	const fileBackendKey = "arco-file-backend"

	// Linux: query D-Bus to find the default Secret Service collection name
	libSecretCollection := getDefaultSecretServiceCollection(log)

	ring, err := keyring.Open(keyring.Config{
		ServiceName: ServiceName,

		// Platform-specific backends will be tried first
		// File backend serves as fallback
		AllowedBackends: []keyring.BackendType{
			keyring.SecretServiceBackend, // Linux (GNOME Keyring, KWallet via Secret Service)
			keyring.KeychainBackend,      // macOS
			keyring.FileBackend,          // Fallback: encrypted file
		},

		// File backend configuration
		FileDir: config.KeyringDir,
		FilePasswordFunc: func(string) (string, error) {
			return fileBackendKey, nil
		},

		// macOS: use default login keychain (empty KeychainName)
		KeychainTrustApplication: true,

		// Linux: use the default collection (queried via D-Bus)
		LibSecretCollectionName: libSecretCollection,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}

	return &Service{
		log:  log,
		ring: ring,
	}, nil
}

// getDefaultSecretServiceCollection queries D-Bus to find the default Secret Service collection name.
// Returns empty string if not on Linux or if query fails.
func getDefaultSecretServiceCollection(log *zap.SugaredLogger) string {
	if !platform.IsLinux() {
		return ""
	}

	conn, err := dbus.SessionBus()
	if err != nil {
		log.Infof("Failed to connect to D-Bus session bus: %v", err)
		return ""
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Infof("Failed to close D-Bus connection: %v", err)
		}
	}()

	obj := conn.Object("org.freedesktop.secrets", "/org/freedesktop/secrets")
	var path dbus.ObjectPath
	err = obj.Call("org.freedesktop.Secret.Service.ReadAlias", 0, "default").Store(&path)
	if err != nil {
		log.Infof("Failed to query default Secret Service collection: %v", err)
		return ""
	}

	// Path format: /org/freedesktop/secrets/collection/<encoded_name>
	// Extract and decode the collection name
	pathStr := string(path)
	prefix := "/org/freedesktop/secrets/collection/"
	if !strings.HasPrefix(pathStr, prefix) {
		return ""
	}

	encodedName := strings.TrimPrefix(pathStr, prefix)
	return decodeSecretServiceName(encodedName)
}

// decodeSecretServiceName decodes Secret Service path encoding (_XX -> character)
func decodeSecretServiceName(encoded string) string {
	// Secret Service encodes special chars as _XX where XX is hex
	// e.g., Default_5fKeyring -> Default_Keyring (_5f = underscore)
	var result strings.Builder
	for i := 0; i < len(encoded); i++ {
		if encoded[i] == '_' && i+2 < len(encoded) {
			hex := encoded[i+1 : i+3]
			if b, err := strconv.ParseUint(hex, 16, 8); err == nil {
				result.WriteByte(byte(b))
				i += 2
				continue
			}
		}
		result.WriteByte(encoded[i])
	}
	return result.String()
}

// GetRepositoryPassword retrieves the password for a repository
func (s *Service) GetRepositoryPassword(repoID int) (string, error) {
	key := fmt.Sprintf(repoPasswordKeyFmt, repoID)
	item, err := s.ring.Get(key)
	if errors.Is(err, keyring.ErrKeyNotFound) {
		// Lets return an empty password so that the user can set a new one when the application rejects access
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("failed to get repository password: %w", err)
	}
	return string(item.Data), nil
}

// SetRepositoryPassword stores the password for a repository
func (s *Service) SetRepositoryPassword(repoID int, password string) error {
	key := fmt.Sprintf(repoPasswordKeyFmt, repoID)
	err := s.ring.Set(keyring.Item{
		Key:  key,
		Data: []byte(password),
	})
	if err != nil {
		return fmt.Errorf("failed to set repository password: %w", err)
	}
	return nil
}

// DeleteRepositoryPassword removes the password for a repository
func (s *Service) DeleteRepositoryPassword(repoID int) error {
	key := fmt.Sprintf(repoPasswordKeyFmt, repoID)
	err := s.ring.Remove(key)
	if err != nil {
		return fmt.Errorf("failed to delete repository password: %w", err)
	}
	return nil
}

// HasRepositoryPassword checks if a password exists for a repository
func (s *Service) HasRepositoryPassword(repoID int) bool {
	key := fmt.Sprintf(repoPasswordKeyFmt, repoID)
	_, err := s.ring.Get(key)
	return err == nil
}

// GetAccessToken retrieves the stored access token
func (s *Service) GetAccessToken() (string, error) {
	s.mu.RLock()
	cached := s.cachedAccessToken
	s.mu.RUnlock()

	if cached != "" {
		return cached, nil
	}

	item, err := s.ring.Get(accessTokenKey)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	s.mu.Lock()
	s.cachedAccessToken = string(item.Data)
	s.mu.Unlock()

	return string(item.Data), nil
}

// SetAccessToken stores the access token
func (s *Service) SetAccessToken(token string) error {
	err := s.ring.Set(keyring.Item{
		Key:  accessTokenKey,
		Data: []byte(token),
	})
	if err != nil {
		return fmt.Errorf("failed to set access token: %w", err)
	}

	s.mu.Lock()
	s.cachedAccessToken = token
	s.mu.Unlock()

	return nil
}

// GetRefreshToken retrieves the stored refresh token
func (s *Service) GetRefreshToken() (string, error) {
	s.mu.RLock()
	cached := s.cachedRefreshToken
	s.mu.RUnlock()

	if cached != "" {
		return cached, nil
	}

	item, err := s.ring.Get(refreshTokenKey)
	if err != nil {
		return "", fmt.Errorf("failed to get refresh token: %w", err)
	}

	s.mu.Lock()
	s.cachedRefreshToken = string(item.Data)
	s.mu.Unlock()

	return string(item.Data), nil
}

// SetRefreshToken stores the refresh token
func (s *Service) SetRefreshToken(token string) error {
	err := s.ring.Set(keyring.Item{
		Key:  refreshTokenKey,
		Data: []byte(token),
	})
	if err != nil {
		return fmt.Errorf("failed to set refresh token: %w", err)
	}

	s.mu.Lock()
	s.cachedRefreshToken = token
	s.mu.Unlock()

	return nil
}

// DeleteTokens removes both access and refresh tokens
func (s *Service) DeleteTokens() error {
	var errs []error

	if err := s.ring.Remove(accessTokenKey); err != nil {
		errs = append(errs, fmt.Errorf("failed to delete access token: %w", err))
	}

	if err := s.ring.Remove(refreshTokenKey); err != nil {
		errs = append(errs, fmt.Errorf("failed to delete refresh token: %w", err))
	}

	// Clear cache regardless of errors
	s.mu.Lock()
	s.cachedAccessToken = ""
	s.cachedRefreshToken = ""
	s.mu.Unlock()

	if len(errs) > 0 {
		return fmt.Errorf("errors deleting tokens: %v", errs)
	}
	return nil
}

// SetTokens stores both access and refresh tokens atomically
func (s *Service) SetTokens(accessToken, refreshToken string) error {
	if err := s.SetAccessToken(accessToken); err != nil {
		return err
	}
	if err := s.SetRefreshToken(refreshToken); err != nil {
		// Try to rollback access token on failure
		if delErr := s.ring.Remove(accessTokenKey); delErr != nil {
			s.log.Warnf("Failed to rollback access token after refresh token error: %v", delErr)
		}
		// Clear access token cache since we removed it from keyring
		s.mu.Lock()
		s.cachedAccessToken = ""
		s.mu.Unlock()
		return err
	}
	return nil
}

// HasRefreshToken checks if a refresh token exists in the keyring
func (s *Service) HasRefreshToken() bool {
	_, err := s.ring.Get(refreshTokenKey)
	return err == nil
}

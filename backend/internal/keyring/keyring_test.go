package keyring

import (
	"testing"

	extKeyring "github.com/99designs/keyring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestLogger() *zap.SugaredLogger {
	logger, _ := zap.NewDevelopment()
	return logger.Sugar()
}

func TestRepositoryPassword(t *testing.T) {
	svc := NewTestService(newTestLogger())

	t.Run("get non-existent password returns empty string", func(t *testing.T) {
		password, err := svc.GetRepositoryPassword(1)
		require.NoError(t, err)
		assert.Empty(t, password)
	})

	t.Run("set and get password", func(t *testing.T) {
		err := svc.SetRepositoryPassword(1, "secret123")
		require.NoError(t, err)

		password, err := svc.GetRepositoryPassword(1)
		require.NoError(t, err)
		assert.Equal(t, "secret123", password)
	})

	t.Run("has password", func(t *testing.T) {
		assert.True(t, svc.HasRepositoryPassword(1))
		assert.False(t, svc.HasRepositoryPassword(999))
	})

	t.Run("update password", func(t *testing.T) {
		err := svc.SetRepositoryPassword(1, "newsecret")
		require.NoError(t, err)

		password, err := svc.GetRepositoryPassword(1)
		require.NoError(t, err)
		assert.Equal(t, "newsecret", password)
	})

	t.Run("delete password", func(t *testing.T) {
		err := svc.DeleteRepositoryPassword(1)
		require.NoError(t, err)

		assert.False(t, svc.HasRepositoryPassword(1))

		password, err := svc.GetRepositoryPassword(1)
		require.NoError(t, err)
		assert.Empty(t, password)
	})

	t.Run("multiple repositories", func(t *testing.T) {
		err := svc.SetRepositoryPassword(10, "pass10")
		require.NoError(t, err)
		err = svc.SetRepositoryPassword(20, "pass20")
		require.NoError(t, err)

		pass10, err := svc.GetRepositoryPassword(10)
		require.NoError(t, err)
		assert.Equal(t, "pass10", pass10)

		pass20, err := svc.GetRepositoryPassword(20)
		require.NoError(t, err)
		assert.Equal(t, "pass20", pass20)
	})
}

func TestAccessToken(t *testing.T) {
	svc := NewTestService(newTestLogger())

	t.Run("get non-existent token returns error", func(t *testing.T) {
		_, err := svc.GetAccessToken()
		assert.Error(t, err)
	})

	t.Run("set and get token", func(t *testing.T) {
		err := svc.SetAccessToken("access123")
		require.NoError(t, err)

		token, err := svc.GetAccessToken()
		require.NoError(t, err)
		assert.Equal(t, "access123", token)
	})

	t.Run("token is cached", func(t *testing.T) {
		// Token should be cached from previous test
		assert.Equal(t, "access123", svc.cachedAccessToken)

		// Get should return cached value
		token, err := svc.GetAccessToken()
		require.NoError(t, err)
		assert.Equal(t, "access123", token)
	})

	t.Run("set updates cache", func(t *testing.T) {
		err := svc.SetAccessToken("newaccess")
		require.NoError(t, err)

		assert.Equal(t, "newaccess", svc.cachedAccessToken)
	})
}

func TestRefreshToken(t *testing.T) {
	svc := NewTestService(newTestLogger())

	t.Run("get non-existent token returns error", func(t *testing.T) {
		_, err := svc.GetRefreshToken()
		assert.Error(t, err)
	})

	t.Run("set and get token", func(t *testing.T) {
		err := svc.SetRefreshToken("refresh123")
		require.NoError(t, err)

		token, err := svc.GetRefreshToken()
		require.NoError(t, err)
		assert.Equal(t, "refresh123", token)
	})

	t.Run("token is cached", func(t *testing.T) {
		assert.Equal(t, "refresh123", svc.cachedRefreshToken)

		token, err := svc.GetRefreshToken()
		require.NoError(t, err)
		assert.Equal(t, "refresh123", token)
	})

	t.Run("has refresh token", func(t *testing.T) {
		assert.True(t, svc.HasRefreshToken())
	})
}

func TestSetTokens(t *testing.T) {
	svc := NewTestService(newTestLogger())

	t.Run("set both tokens", func(t *testing.T) {
		err := svc.SetTokens("access", "refresh")
		require.NoError(t, err)

		accessToken, err := svc.GetAccessToken()
		require.NoError(t, err)
		assert.Equal(t, "access", accessToken)

		refreshToken, err := svc.GetRefreshToken()
		require.NoError(t, err)
		assert.Equal(t, "refresh", refreshToken)
	})
}

func TestDeleteTokens(t *testing.T) {
	svc := NewTestService(newTestLogger())

	t.Run("delete clears both tokens and cache", func(t *testing.T) {
		// Set tokens first
		err := svc.SetTokens("access", "refresh")
		require.NoError(t, err)

		// Delete tokens
		err = svc.DeleteTokens()
		require.NoError(t, err)

		// Cache should be cleared
		assert.Empty(t, svc.cachedAccessToken)
		assert.Empty(t, svc.cachedRefreshToken)

		// HasRefreshToken should return false
		assert.False(t, svc.HasRefreshToken())
	})

	t.Run("delete on empty keyring clears cache", func(t *testing.T) {
		svc2 := NewTestService(newTestLogger())
		svc2.cachedAccessToken = "stale"
		svc2.cachedRefreshToken = "stale"

		// Delete should clear cache even if tokens don't exist
		_ = svc2.DeleteTokens()

		assert.Empty(t, svc2.cachedAccessToken)
		assert.Empty(t, svc2.cachedRefreshToken)
	})
}

func TestDecodeSecretServiceName(t *testing.T) {
	tests := []struct {
		name     string
		encoded  string
		expected string
	}{
		{
			name:     "no encoding needed",
			encoded:  "login",
			expected: "login",
		},
		{
			name:     "underscore encoded",
			encoded:  "Default_5fKeyring",
			expected: "Default_Keyring",
		},
		{
			name:     "space encoded",
			encoded:  "My_20Collection",
			expected: "My Collection",
		},
		{
			name:     "multiple encodings",
			encoded:  "Test_5fName_20With_5fSpaces",
			expected: "Test_Name With_Spaces",
		},
		{
			name:     "invalid hex preserved",
			encoded:  "Test_ZZValue",
			expected: "Test_ZZValue",
		},
		{
			name:     "underscore at end preserved",
			encoded:  "Test_",
			expected: "Test_",
		},
		{
			name:     "short trailing underscore",
			encoded:  "Test_5",
			expected: "Test_5",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := decodeSecretServiceName(tc.encoded)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTokenCacheInvalidation(t *testing.T) {
	svc := NewTestService(newTestLogger())

	t.Run("cache populated on get", func(t *testing.T) {
		// Manually set in keyring without going through Set
		_ = svc.ring.Set(extKeyring.Item{Key: accessTokenKey, Data: []byte("direct")})

		// Cache should be empty
		assert.Empty(t, svc.cachedAccessToken)

		// Get should populate cache
		token, err := svc.GetAccessToken()
		require.NoError(t, err)
		assert.Equal(t, "direct", token)
		assert.Equal(t, "direct", svc.cachedAccessToken)
	})

	t.Run("set overwrites cache", func(t *testing.T) {
		svc.cachedAccessToken = "old"

		err := svc.SetAccessToken("new")
		require.NoError(t, err)

		assert.Equal(t, "new", svc.cachedAccessToken)
	})
}

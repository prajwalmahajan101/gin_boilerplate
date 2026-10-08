package auth

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
)

func testTokenService() *TokenService {
	return NewTokenService(&config.Config{
		AuthTokenSecret:  "test-secret-key-32-bytes-long!!!",
		AuthTokenTTLMin:  15,
		RefreshTokenTTLH: 168,
	})
}

func TestGenerateAndParseAccess(t *testing.T) {
	ts := testTokenService()
	tok, err := ts.GenerateAccess(42, "admin")
	require.NoError(t, err)

	claims, err := ts.ParseAccess(tok)
	require.NoError(t, err)
	require.Equal(t, int64(42), claims.UserID)
	require.Equal(t, "admin", claims.Role)
}

func TestParseAccess_RejectsRefreshToken(t *testing.T) {
	ts := testTokenService()
	refresh, err := ts.GenerateRefresh(42)
	require.NoError(t, err)

	_, err = ts.ParseAccess(refresh)
	require.Error(t, err)
}

func TestGenerateAndParseRefresh(t *testing.T) {
	ts := testTokenService()
	tok, err := ts.GenerateRefresh(99)
	require.NoError(t, err)

	uid, err := ts.ParseRefresh(tok)
	require.NoError(t, err)
	require.Equal(t, int64(99), uid)
}

func TestParseRefresh_RejectsAccessToken(t *testing.T) {
	ts := testTokenService()
	access, err := ts.GenerateAccess(99, "user")
	require.NoError(t, err)

	_, err = ts.ParseRefresh(access)
	require.Error(t, err)
}

func TestGeneratePair(t *testing.T) {
	ts := testTokenService()
	pair, err := ts.GeneratePair(7, "user")
	require.NoError(t, err)
	require.NotEmpty(t, pair.AccessToken)
	require.NotEmpty(t, pair.RefreshToken)
}

func TestParse_BadSignature(t *testing.T) {
	ts := testTokenService()
	tok, err := ts.GenerateAccess(1, "user")
	require.NoError(t, err)

	other := NewTokenService(&config.Config{
		AuthTokenSecret:  "different-secret-key-32-bytes!!!",
		AuthTokenTTLMin:  15,
		RefreshTokenTTLH: 168,
	})
	_, err = other.ParseAccess(tok)
	require.Error(t, err)
}

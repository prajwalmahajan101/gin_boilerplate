package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashAndCheck(t *testing.T) {
	hash, err := HashPassword("s3cret!")
	require.NoError(t, err)
	require.NotEmpty(t, hash)
	require.True(t, CheckPassword(hash, "s3cret!"))
	require.False(t, CheckPassword(hash, "wrong"))
}

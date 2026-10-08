package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseKey_Valid(t *testing.T) {
	prefix := randomHex(4)
	secret := randomHex(16)
	raw := "gbp_" + prefix + "." + secret

	p, s, ok := parseKey(raw)
	require.True(t, ok)
	require.Equal(t, prefix, p)
	require.Equal(t, secret, s)
}

func TestParseKey_Invalid(t *testing.T) {
	cases := []string{
		"",
		"not-a-key",
		"gbp_short.short",
		"gbp_12345678",
		"xxx_12345678.12345678901234567890123456789012",
	}
	for _, c := range cases {
		_, _, ok := parseKey(c)
		require.False(t, ok, "expected false for %q", c)
	}
}

func TestHashDeterministic(t *testing.T) {
	svc := &APIKeyService{pepper: "test-pepper"}
	h1 := svc.hash("secret123")
	h2 := svc.hash("secret123")
	require.Equal(t, h1, h2)
	require.NotEqual(t, h1, svc.hash("different"))
}

func TestRandomHex_Length(t *testing.T) {
	h := randomHex(4)
	require.Len(t, h, 8)
	h = randomHex(16)
	require.Len(t, h, 32)
}

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/valkey"
)

// EmailSender delivers transactional email. Satisfied by aws.SES in production
// and a fake in tests, so the auth service never imports the AWS SDK directly.
type EmailSender interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

// resetStore persists single-use password-reset tokens and per-user revocation
// epochs. Backed by Valkey in production (ResetStore); faked in tests.
type resetStore interface {
	// PutToken maps a token hash to a user id for ttl.
	PutToken(ctx context.Context, hash string, uid int64, ttl time.Duration) error
	// TakeToken returns the user id for hash and consumes it (single-use).
	TakeToken(ctx context.Context, hash string) (uid int64, found bool, err error)
	// SetEpoch records "all tokens issued before now are revoked" for uid.
	SetEpoch(ctx context.Context, uid int64, ttl time.Duration) error
}

// ResetStore is the Valkey-backed resetStore. It also serves the user-epoch
// lookup consumed by the auth middleware.
type ResetStore struct {
	vk *valkey.Client
}

func NewResetStore(vk *valkey.Client) *ResetStore { return &ResetStore{vk: vk} }

func tokenKey(hash string) string { return "pwr:" + hash }
func epochKey(uid int64) string   { return "uepoch:" + strconv.FormatInt(uid, 10) }

func (s *ResetStore) PutToken(ctx context.Context, hash string, uid int64, ttl time.Duration) error {
	return s.vk.SetEX(ctx, tokenKey(hash), strconv.FormatInt(uid, 10), ttl)
}

func (s *ResetStore) TakeToken(ctx context.Context, hash string) (int64, bool, error) {
	v, err := s.vk.Get(ctx, tokenKey(hash))
	if err != nil {
		return 0, false, err
	}
	if v == "" {
		return 0, false, nil
	}
	_ = s.vk.Del(ctx, tokenKey(hash)) // single-use: best-effort consume
	uid, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("reset: corrupt token value: %w", err)
	}
	return uid, true, nil
}

func (s *ResetStore) SetEpoch(ctx context.Context, uid int64, ttl time.Duration) error {
	return s.vk.SetEX(ctx, epochKey(uid), strconv.FormatInt(time.Now().Unix(), 10), ttl)
}

// Epoch returns the revocation epoch (unix seconds) for uid, or 0 if none. This
// is the middleware.UserEpochChecker.
func (s *ResetStore) Epoch(ctx context.Context, uid int64) int64 {
	v, err := s.vk.Get(ctx, epochKey(uid))
	if err != nil || v == "" {
		return 0
	}
	ts, _ := strconv.ParseInt(v, 10, 64)
	return ts
}

// newResetToken returns a URL-safe random token and its hex SHA-256 (stored at
// rest so a leaked store cannot be used to reset passwords).
func newResetToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("reset: read random: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

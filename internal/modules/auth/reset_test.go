package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
)

type fakeStore struct {
	putHash  string
	putUID   int64
	takeUID  int64
	found    bool
	takeErr  error
	tookHash string
	epochUID int64
	epochTTL time.Duration
	epochSet bool
}

func (f *fakeStore) PutToken(_ context.Context, hash string, uid int64, _ time.Duration) error {
	f.putHash, f.putUID = hash, uid
	return nil
}

func (f *fakeStore) TakeToken(_ context.Context, hash string) (int64, bool, error) {
	f.tookHash = hash
	return f.takeUID, f.found, f.takeErr
}

func (f *fakeStore) SetEpoch(_ context.Context, uid int64, ttl time.Duration) error {
	f.epochUID, f.epochTTL, f.epochSet = uid, ttl, true
	return nil
}

func TestHashToken_Deterministic(t *testing.T) {
	a, b := hashToken("abc"), hashToken("abc")
	if a != b {
		t.Fatal("hashToken must be deterministic")
	}
	if a == hashToken("abd") {
		t.Fatal("different tokens must hash differently")
	}
}

func TestNewResetToken_UniqueAndHashed(t *testing.T) {
	t1, h1, err := newResetToken()
	if err != nil {
		t.Fatalf("newResetToken: %v", err)
	}
	t2, _, _ := newResetToken()
	if t1 == t2 {
		t.Fatal("tokens must be unique")
	}
	if h1 != hashToken(t1) {
		t.Fatal("returned hash must equal hashToken(token)")
	}
}

func TestResetEmailBody(t *testing.T) {
	raw := (&AuthService{}).resetEmailBody("TKN")
	if !strings.Contains(raw, "TKN") || strings.Contains(raw, "http") {
		t.Fatalf("raw body should contain the token, no URL: %q", raw)
	}
	linked := (&AuthService{resetURLBase: "https://app/reset"}).resetEmailBody("TKN")
	if !strings.Contains(linked, "https://app/reset?token=TKN") {
		t.Fatalf("linked body should embed the reset URL: %q", linked)
	}
}

func TestRequestPasswordReset_NotConfigured(t *testing.T) {
	err := (&AuthService{}).RequestPasswordReset(context.Background(), "a@b.com")
	if !apperr.Is(err, apperr.CodeServiceUnavail) {
		t.Fatalf("err=%v, want service unavailable", err)
	}
}

func TestResetPassword_NotConfigured(t *testing.T) {
	err := (&AuthService{}).ResetPassword(context.Background(), "tok", "password123")
	if !apperr.Is(err, apperr.CodeServiceUnavail) {
		t.Fatalf("err=%v, want service unavailable", err)
	}
}

func TestResetPassword_InvalidToken(t *testing.T) {
	fs := &fakeStore{found: false}
	err := (&AuthService{reset: fs}).ResetPassword(context.Background(), "tok", "password123")
	if !apperr.Is(err, apperr.CodeValidationError) {
		t.Fatalf("err=%v, want validation error", err)
	}
	if fs.tookHash != hashToken("tok") {
		t.Fatalf("TakeToken got %q, want hash of token", fs.tookHash)
	}
}

func TestResetPassword_StoreErrorPropagates(t *testing.T) {
	sentinel := errors.New("valkey down")
	fs := &fakeStore{takeErr: sentinel}
	err := (&AuthService{reset: fs}).ResetPassword(context.Background(), "tok", "password123")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err=%v, want wrapped store error", err)
	}
}

func TestRevokeSessions_SetsEpochWithRefreshTTL(t *testing.T) {
	fs := &fakeStore{}
	s := &AuthService{reset: fs, tokens: &TokenService{refreshTTL: 72 * time.Hour}}
	s.revokeSessions(context.Background(), 42)
	if !fs.epochSet || fs.epochUID != 42 || fs.epochTTL != 72*time.Hour {
		t.Fatalf("epoch not set correctly: set=%v uid=%d ttl=%v", fs.epochSet, fs.epochUID, fs.epochTTL)
	}
}

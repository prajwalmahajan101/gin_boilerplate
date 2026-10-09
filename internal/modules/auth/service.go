package auth

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store"
)

type AuthService struct {
	*store.BaseService[User]
	repo      *Repository
	tokens    *TokenService
	blacklist *Blacklist

	// Password-reset deps (optional; nil sender/reset disables the flow).
	sender       EmailSender
	reset        resetStore
	resetTTL     time.Duration
	resetURLBase string
}

// ResetDeps bundles the optional password-reset collaborators.
type ResetDeps struct {
	Sender  EmailSender
	Store   resetStore
	TTL     time.Duration
	URLBase string
}

func NewService(pool *pgxpool.Pool, tokens *TokenService, blacklist *Blacklist, reset ResetDeps) *AuthService {
	repo := NewRepository(pool)
	base := store.NewBaseService[User](repo.Repository, pool, authHooks{repo: repo})
	return &AuthService{
		BaseService:  base,
		repo:         repo,
		tokens:       tokens,
		blacklist:    blacklist,
		sender:       reset.Sender,
		reset:        reset.Store,
		resetTTL:     reset.TTL,
		resetURLBase: reset.URLBase,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (TokenPair, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return TokenPair{}, apperr.InternalError(err)
	}
	u := &User{
		Email:        email,
		PasswordHash: hash,
		Role:         RoleUser,
	}
	if err := s.Create(ctx, u); err != nil {
		return TokenPair{}, err
	}
	return s.tokens.GeneratePair(u.ID, u.Role)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (TokenPair, error) {
	row, found, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return TokenPair{}, err
	}
	if !found || !CheckPassword(row.PasswordHash, password) {
		return TokenPair{}, apperr.Unauthorized("invalid credentials")
	}
	if !row.IsActive {
		return TokenPair{}, apperr.Unauthorized("account disabled")
	}
	_ = s.repo.TouchLogin(ctx, row.ID)
	return s.tokens.GeneratePair(row.ID, row.Role)
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (TokenPair, error) {
	rc, err := s.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return TokenPair{}, apperr.Unauthorized("invalid refresh token")
	}
	if rc.JTI != "" && s.blacklist.IsBlacklisted(ctx, rc.JTI) {
		return TokenPair{}, apperr.Unauthorized("token revoked")
	}
	u, err := s.GetByIDOrFail(ctx, rc.UserID)
	if err != nil {
		return TokenPair{}, err
	}
	if !u.IsActive {
		return TokenPair{}, apperr.Unauthorized("account disabled")
	}
	return s.tokens.GeneratePair(u.ID, u.Role)
}

func (s *AuthService) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	u, err := s.GetByIDOrFail(ctx, userID)
	if err != nil {
		return err
	}
	if !CheckPassword(u.PasswordHash, oldPassword) {
		return apperr.Unauthorized("incorrect current password")
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return apperr.InternalError(err)
	}
	u.PasswordHash = hash
	if err := s.Update(ctx, &u); err != nil {
		return err
	}
	s.revokeSessions(ctx, userID)
	return nil
}

// RequestPasswordReset issues a reset token and emails it. It never reveals
// whether the email belongs to a real account (no user enumeration): an unknown
// email returns nil without sending anything.
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) error {
	if s.sender == nil || s.reset == nil {
		return apperr.ServiceUnavailable("password reset not configured")
	}
	row, found, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return err
	}
	if !found || !row.IsActive {
		return nil
	}
	token, hash, err := newResetToken()
	if err != nil {
		return apperr.InternalError(err)
	}
	if err := s.reset.PutToken(ctx, hash, row.ID, s.resetTTL); err != nil {
		return err
	}
	body := s.resetEmailBody(token)
	if err := s.sender.SendEmail(ctx, email, "Password reset", body); err != nil {
		slog.ErrorContext(ctx, "password reset email failed", slog.Int64("user_id", row.ID), slog.Any("error", err))
		return apperr.InternalError(err)
	}
	return nil
}

// ResetPassword consumes a reset token, sets the new password, and revokes all
// of the user's existing sessions.
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if s.reset == nil {
		return apperr.ServiceUnavailable("password reset not configured")
	}
	uid, found, err := s.reset.TakeToken(ctx, hashToken(token))
	if err != nil {
		return err
	}
	if !found {
		return apperr.ValidationError("invalid or expired reset token")
	}
	u, err := s.GetByIDOrFail(ctx, uid)
	if err != nil {
		return err
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return apperr.InternalError(err)
	}
	u.PasswordHash = hash
	if err := s.Update(ctx, &u); err != nil {
		return err
	}
	s.revokeSessions(ctx, uid)
	return nil
}

// revokeSessions marks all tokens issued before now as revoked for uid. The
// epoch outlives the longest-lived (refresh) token, after which it is moot.
func (s *AuthService) revokeSessions(ctx context.Context, uid int64) {
	if s.reset == nil {
		return
	}
	if err := s.reset.SetEpoch(ctx, uid, s.tokens.RefreshTTL()); err != nil {
		slog.ErrorContext(ctx, "revoke sessions failed", slog.Int64("user_id", uid), slog.Any("error", err))
	}
}

func (s *AuthService) resetEmailBody(token string) string {
	if s.resetURLBase != "" {
		return "Reset your password: " + s.resetURLBase + "?token=" + token
	}
	return "Your password reset token is:\n\n" + token + "\n\nPOST it with your new password to /api/v1/auth/reset-password."
}

func (s *AuthService) Logout(ctx context.Context, accessJTI, refreshJTI string) error {
	if err := s.blacklist.Add(ctx, accessJTI, s.tokens.AccessTTL()); err != nil {
		return err
	}
	return s.blacklist.Add(ctx, refreshJTI, s.tokens.RefreshTTL())
}

type authHooks struct {
	store.NoOpHooks[User]
	repo *Repository
}

func (h authHooks) PreCreate(ctx context.Context, m *User) error {
	_, found, err := h.repo.GetByEmail(ctx, m.Email)
	if err != nil {
		return err
	}
	if found {
		return apperr.Conflict("email already registered")
	}
	return nil
}

package auth

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store"
)

type AuthService struct {
	*store.BaseService[User]
	repo   *Repository
	tokens *TokenService
}

func NewService(pool *pgxpool.Pool, tokens *TokenService) *AuthService {
	repo := NewRepository(pool)
	base := store.NewBaseService[User](repo.Repository, pool, authHooks{repo: repo})
	return &AuthService{BaseService: base, repo: repo, tokens: tokens}
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
	uid, err := s.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return TokenPair{}, apperr.Unauthorized("invalid refresh token")
	}
	u, err := s.GetByIDOrFail(ctx, uid)
	if err != nil {
		return TokenPair{}, err
	}
	if !u.IsActive {
		return TokenPair{}, apperr.Unauthorized("account disabled")
	}
	return s.tokens.GeneratePair(u.ID, u.Role)
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

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store/db"
)

type APIKey struct {
	store.BaseModel
	UserID     int64      `db:"user_id"`
	Name       string     `db:"name"`
	Prefix     string     `db:"prefix"`
	KeyHash    string     `db:"key_hash"`
	ExpiresAt  *time.Time `db:"expires_at"`
	LastUsedAt *time.Time `db:"last_used_at"`
}

func (APIKey) TableName() string { return "api_keys" }

type APIKeyResult struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Prefix    string     `json:"prefix"`
	Key       string     `json:"key,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type APIKeyService struct {
	repo   *store.Repository[APIKey]
	q      *db.Queries
	pepper string
}

func NewAPIKeyService(pool *pgxpool.Pool, pepper string) *APIKeyService {
	return &APIKeyService{
		repo:   store.NewRepository[APIKey](pool),
		q:      db.New(pool),
		pepper: pepper,
	}
}

func (s *APIKeyService) Generate(ctx context.Context, userID int64, name string, expiresAt *time.Time) (APIKeyResult, error) {
	prefix := randomHex(4)
	secret := randomHex(16)
	raw := fmt.Sprintf("gbp_%s.%s", prefix, secret)
	hash := s.hash(secret)

	k := &APIKey{
		UserID:    userID,
		Name:      name,
		Prefix:    prefix,
		KeyHash:   hash,
		ExpiresAt: expiresAt,
	}
	if err := s.repo.Create(ctx, k); err != nil {
		return APIKeyResult{}, err
	}
	return APIKeyResult{
		ID:        k.ID,
		Name:      k.Name,
		Prefix:    k.Prefix,
		Key:       raw,
		ExpiresAt: k.ExpiresAt,
		CreatedAt: k.CreatedAt,
	}, nil
}

func (s *APIKeyService) Validate(ctx context.Context, rawKey string) (userID int64, role string, err error) {
	prefix, secret, ok := parseKey(rawKey)
	if !ok {
		return 0, "", apperr.Unauthorized("malformed api key")
	}
	row, qerr := s.q.GetAPIKeyByPrefix(ctx, prefix)
	if errors.Is(qerr, pgx.ErrNoRows) {
		return 0, "", apperr.Unauthorized("invalid api key")
	}
	if qerr != nil {
		return 0, "", qerr
	}
	if row.KeyHash != s.hash(secret) {
		return 0, "", apperr.Unauthorized("invalid api key")
	}
	if row.ExpiresAt.Valid && row.ExpiresAt.Time.Before(time.Now()) {
		return 0, "", apperr.Unauthorized("api key expired")
	}
	_ = s.q.TouchAPIKeyUsage(ctx, row.ID)
	return row.UserID, row.UserRole, nil
}

func (s *APIKeyService) ListByUser(ctx context.Context, userID int64) ([]APIKeyResult, error) {
	rows, err := s.q.ListAPIKeysByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]APIKeyResult, len(rows))
	for i := range rows {
		out[i] = APIKeyResult{
			ID:        rows[i].ID,
			Name:      rows[i].Name,
			Prefix:    rows[i].Prefix,
			CreatedAt: rows[i].CreatedAt.Time,
		}
		if rows[i].ExpiresAt.Valid {
			t := rows[i].ExpiresAt.Time
			out[i].ExpiresAt = &t
		}
	}
	return out, nil
}

func (s *APIKeyService) Revoke(ctx context.Context, id int64) error {
	return s.repo.SoftDelete(ctx, id)
}

func (s *APIKeyService) hash(secret string) string {
	h := sha256.Sum256([]byte(secret + s.pepper))
	return hex.EncodeToString(h[:])
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func parseKey(raw string) (prefix, secret string, ok bool) {
	raw, ok = strings.CutPrefix(raw, "gbp_")
	if !ok {
		return "", "", false
	}
	parts := strings.SplitN(raw, ".", 2)
	if len(parts) != 2 || len(parts[0]) != 8 || len(parts[1]) != 32 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

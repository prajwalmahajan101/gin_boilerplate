package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
)

type Claims struct {
	UserID int64  `json:"uid"`
	Role   string `json:"role"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type TokenService struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenService(cfg *config.Config) *TokenService {
	return &TokenService{
		secret:     []byte(cfg.AuthTokenSecret),
		accessTTL:  time.Duration(cfg.AuthTokenTTLMin) * time.Minute,
		refreshTTL: time.Duration(cfg.RefreshTokenTTLH) * time.Hour,
	}
}

func (ts *TokenService) GenerateAccess(userID int64, role string) (string, error) {
	return ts.sign(jwt.MapClaims{
		"uid":  userID,
		"role": role,
		"typ":  "access",
		"exp":  jwt.NewNumericDate(time.Now().Add(ts.accessTTL)),
		"iat":  jwt.NewNumericDate(time.Now()),
	})
}

func (ts *TokenService) GenerateRefresh(userID int64) (string, error) {
	return ts.sign(jwt.MapClaims{
		"uid": userID,
		"typ": "refresh",
		"exp": jwt.NewNumericDate(time.Now().Add(ts.refreshTTL)),
		"iat": jwt.NewNumericDate(time.Now()),
	})
}

func (ts *TokenService) GeneratePair(userID int64, role string) (TokenPair, error) {
	access, err := ts.GenerateAccess(userID, role)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := ts.GenerateRefresh(userID)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func (ts *TokenService) ParseAccess(tokenStr string) (Claims, error) {
	claims, err := ts.parse(tokenStr)
	if err != nil {
		return Claims{}, err
	}
	if claims["typ"] != "access" {
		return Claims{}, fmt.Errorf("token: expected access token")
	}
	return extractClaims(claims)
}

func (ts *TokenService) ParseRefresh(tokenStr string) (int64, error) {
	claims, err := ts.parse(tokenStr)
	if err != nil {
		return 0, err
	}
	if claims["typ"] != "refresh" {
		return 0, fmt.Errorf("token: expected refresh token")
	}
	uid, ok := claims["uid"].(float64)
	if !ok {
		return 0, fmt.Errorf("token: invalid uid")
	}
	return int64(uid), nil
}

func (ts *TokenService) sign(claims jwt.MapClaims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(ts.secret)
}

func (ts *TokenService) parse(tokenStr string) (jwt.MapClaims, error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("token: unexpected signing method %v", t.Header["alg"])
		}
		return ts.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok || !tok.Valid {
		return nil, fmt.Errorf("token: invalid")
	}
	return claims, nil
}

func (ts *TokenService) AccessParser() func(string) (int64, string, error) {
	return func(tokenStr string) (int64, string, error) {
		c, err := ts.ParseAccess(tokenStr)
		if err != nil {
			return 0, "", err
		}
		return c.UserID, c.Role, nil
	}
}

func extractClaims(m jwt.MapClaims) (Claims, error) {
	uid, ok := m["uid"].(float64)
	if !ok {
		return Claims{}, fmt.Errorf("token: invalid uid")
	}
	role, _ := m["role"].(string)
	return Claims{UserID: int64(uid), Role: role}, nil
}

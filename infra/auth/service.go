package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/KNattawat89/hospital-middleware-system/infra/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// ContextSubjectKey is the gin.Context key the token subject (e.g. user/staff
// ID) is stored under once a request passes authorization.
const ContextSubjectKey = "auth_subject"

type Claims struct {
	jwt.RegisteredClaims
	Type TokenType `json:"typ"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// AuthorizeResult is returned by Authorize. NewTokens is non-nil only when
// the access token had expired and was transparently renewed via the refresh token.
type AuthorizeResult struct {
	Subject   string
	NewTokens *TokenPair
}

// Service validates and issues JWT access/refresh tokens.
type Service struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewService(cfg *config.Config) (*Service, error) {
	if cfg.Jwt.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	accessTTL, err := time.ParseDuration(cfg.Jwt.AccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("parse JWT_ACCESS_TOKEN_TTL: %w", err)
	}

	refreshTTL, err := time.ParseDuration(cfg.Jwt.RefreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("parse JWT_REFRESH_TOKEN_TTL: %w", err)
	}

	return &Service{
		secret:     []byte(cfg.Jwt.Secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}, nil
}

// GenerateTokenPair issues a fresh access/refresh token pair for the given
// subject (typically a user/staff ID).
func (s *Service) GenerateTokenPair(subject string) (*TokenPair, error) {
	access, err := s.sign(subject, TokenTypeAccess, s.accessTTL)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refresh, err := s.sign(subject, TokenTypeRefresh, s.refreshTTL)
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	return &TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

// Authorize validates the access token. If it's expired (any other failure
// is rejected outright, not just any error), it falls back to validating the
// refresh token and, on success, transparently mints a new token pair.
func (s *Service) Authorize(accessToken, refreshToken string) (*AuthorizeResult, error) {
	claims, err := s.parse(accessToken, TokenTypeAccess)
	if err == nil {
		return &AuthorizeResult{Subject: claims.Subject}, nil
	}

	if !errors.Is(err, jwt.ErrTokenExpired) {
		return nil, err
	}

	refreshClaims, err := s.parse(refreshToken, TokenTypeRefresh)
	if err != nil {
		return nil, fmt.Errorf("refresh token invalid: %w", err)
	}

	pair, err := s.GenerateTokenPair(refreshClaims.Subject)
	if err != nil {
		return nil, err
	}

	return &AuthorizeResult{Subject: refreshClaims.Subject, NewTokens: pair}, nil
}

// Middleware is Gin middleware guarding routes behind a valid (or
// transparently renewable) access token. The access token is read from the
// `Authorization: Bearer <token>` header and the refresh token from
// `X-Refresh-Token`. When the access token had to be renewed, the new pair
// is echoed back via `X-Access-Token` / `X-Refresh-Token` response headers.
func (s *Service) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || accessToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "missing bearer token"})
			return
		}

		result, err := s.Authorize(accessToken, c.GetHeader("X-Refresh-Token"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid or expired token"})
			return
		}

		if result.NewTokens != nil {
			c.Header("X-Access-Token", result.NewTokens.AccessToken)
			c.Header("X-Refresh-Token", result.NewTokens.RefreshToken)
		}

		c.Set(ContextSubjectKey, result.Subject)
		c.Next()
	}
}

func (s *Service) sign(subject string, typ TokenType, ttl time.Duration) (string, error) {
	now := time.Now()

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Type: typ,
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

func (s *Service) parse(tokenString string, want TokenType) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	if claims.Type != want {
		return nil, fmt.Errorf("unexpected token type: %s", claims.Type)
	}

	return claims, nil
}

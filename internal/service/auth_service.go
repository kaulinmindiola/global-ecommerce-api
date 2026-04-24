package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/repository"
)

// AuthService defines JWT token lifecycle operations.
// Separated from UserService so authentication infrastructure concerns
// (signing, revocation) stay decoupled from user business logic.
type AuthService interface {
	// IssueTokens generates a new access + refresh token pair for a user.
	IssueTokens(ctx context.Context, userID uuid.UUID) (*TokenPairResult, error)

	// ValidateAccessToken parses and verifies a JWT access token.
	// Returns the claims on success or an error if invalid/expired.
	ValidateAccessToken(ctx context.Context, rawToken string) (*Claims, error)

	// RefreshTokens validates a refresh token and issues a new token pair.
	// The old refresh token is invalidated (rotation strategy).
	RefreshTokens(ctx context.Context, refreshToken string) (*TokenPairResult, error)

	// RevokeToken adds an access token to the blocklist in Redis.
	// Used by the logout endpoint.
	RevokeToken(ctx context.Context, rawToken string) error
}

// TokenPairResult is the output DTO for any token-issuing operation.
type TokenPairResult struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"` // seconds
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Claims holds the JWT payload fields for our application.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// authService is the concrete JWT implementation of AuthService.
type authService struct {
	secretKey       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	cache           repository.CacheRepository
}

// NewAuthService constructs an AuthService with the given signing secret.
// secretKey should be loaded from environment variables, never hardcoded.
func NewAuthService(secretKey string, cache repository.CacheRepository) AuthService {
	return &authService{
		secretKey:       []byte(secretKey),
		accessTokenTTL:  24 * time.Hour,
		refreshTokenTTL: 7 * 24 * time.Hour,
		cache:           cache,
	}
}

// IssueTokens generates a signed access token and a refresh token for a user.
func (s *authService) IssueTokens(_ context.Context, userID uuid.UUID) (*TokenPairResult, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.accessTokenTTL)

	// Build access token claims.
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        uuid.New().String(), // jti — unique token identifier for blocklisting
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secretKey)
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	// Refresh token is a plain signed JWT with a longer TTL.
	// In production, consider using opaque tokens stored in Redis instead.
	refreshClaims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTokenTTL)),
			ID:        uuid.New().String(),
		},
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(s.secretKey)
	if err != nil {
		return nil, fmt.Errorf("signing refresh token: %w", err)
	}

	return &TokenPairResult{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.accessTokenTTL.Seconds()),
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// ValidateAccessToken parses and verifies a JWT access token.
// Checks: signature validity, expiry, and blocklist membership.
func (s *authService) ValidateAccessToken(ctx context.Context, rawToken string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(rawToken, &Claims{}, func(t *jwt.Token) (any, error) {
		// Enforce the expected signing algorithm to prevent alg=none attacks.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secretKey, nil
	})
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, domain.ErrUnauthorized
	}

	// Check if this specific token JTI is on the blocklist (logged out).
	blocklisted, err := s.cache.Exists(ctx, blockedTokenKey(claims.ID))
	if err == nil && blocklisted {
		return nil, domain.ErrUnauthorized
	}

	return claims, nil
}

// RefreshTokens validates a refresh token and issues a new token pair.
func (s *authService) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPairResult, error) {
	claims, err := s.ValidateAccessToken(ctx, refreshToken)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	// Revoke the old refresh token immediately (rotation — prevents replay attacks).
	_ = s.cache.Set(ctx, blockedTokenKey(claims.ID), true, s.refreshTokenTTL)

	return s.IssueTokens(ctx, claims.UserID)
}

// RevokeToken adds the token's JTI to the Redis blocklist with its remaining TTL.
// After this, ValidateAccessToken will reject the token even if the signature is valid.
func (s *authService) RevokeToken(ctx context.Context, rawToken string) error {
	token, err := jwt.ParseWithClaims(rawToken, &Claims{}, func(t *jwt.Token) (any, error) {
		return s.secretKey, nil
	})
	if err != nil {
		// If we cannot parse the token, there is nothing to revoke — treat as success.
		return nil
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil
	}

	// TTL for the blocklist entry matches the token's remaining lifetime.
	remaining := time.Until(claims.ExpiresAt.Time)
	if remaining <= 0 {
		// Token is already expired — no need to blocklist it.
		return nil
	}

	return s.cache.Set(ctx, blockedTokenKey(claims.ID), true, remaining)
}

// blockedTokenKey generates the Redis key for a blocked token JTI.
func blockedTokenKey(jti string) string {
	return fmt.Sprintf("auth:blocked:%s", jti)
}

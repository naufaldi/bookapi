package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"bookapi/internal/platform/crypto" // JWT/Password helpers
	"bookapi/internal/session"
	"bookapi/internal/user"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
)

type Service struct {
	secret         string
	userService    *user.Service
	sessionService *session.Service
}

func NewService(secret string, userService *user.Service, sessionService *session.Service) *Service {
	return &Service{
		secret:         secret,
		userService:    userService,
		sessionService: sessionService,
	}
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *Service) Login(ctx context.Context, email, password string, rememberMe bool, userAgent, ipAddress string) (string, string, int, error) {
	u, err := s.userService.GetByEmail(ctx, email)
	if err != nil || !crypto.VerifyPassword(u.Password, password) {
		return "", "", 0, ErrUnauthorized
	}

	accessTokenTTL := 15 * time.Minute
	refreshTokenTTL := 30 * 24 * time.Hour
	if rememberMe {
		refreshTokenTTL = 90 * 24 * time.Hour
	}

	accessToken, _, err := crypto.GenerateToken(s.secret, u.ID, u.Role, accessTokenTTL)
	if err != nil {
		return "", "", 0, err
	}

	refreshTokenBytes := make([]byte, 32)
	if _, err := rand.Read(refreshTokenBytes); err != nil {
		return "", "", 0, err
	}
	refreshToken := hex.EncodeToString(refreshTokenBytes)
	tokenHash := hashToken(refreshToken)

	sess := &session.Session{
		UserID:           u.ID,
		RefreshTokenHash: tokenHash,
		UserAgent:        userAgent,
		IPAddress:        ipAddress,
		RememberMe:       rememberMe,
		ExpiresAt:        time.Now().Add(refreshTokenTTL),
	}

	if err := s.sessionService.Create(ctx, sess); err != nil {
		return "", "", 0, err
	}

	return accessToken, refreshToken, int(accessTokenTTL.Seconds()), nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (string, string, int, error) {
	tokenHash := hashToken(refreshToken)
	sess, err := s.sessionService.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return "", "", 0, ErrUnauthorized
	}

	u, err := s.userService.GetByID(ctx, sess.UserID)
	if err != nil {
		return "", "", 0, ErrUnauthorized
	}

	if err := s.sessionService.DeleteByTokenHash(ctx, tokenHash); err != nil {
		return "", "", 0, err
	}

	accessTokenTTL := 15 * time.Minute
	refreshTokenTTL := 30 * 24 * time.Hour
	if sess.RememberMe {
		refreshTokenTTL = 90 * 24 * time.Hour
	}

	accessToken, _, err := crypto.GenerateToken(s.secret, u.ID, u.Role, accessTokenTTL)
	if err != nil {
		return "", "", 0, err
	}

	refreshTokenBytes := make([]byte, 32)
	if _, err := rand.Read(refreshTokenBytes); err != nil {
		return "", "", 0, err
	}
	newRefreshToken := hex.EncodeToString(refreshTokenBytes)
	newTokenHash := hashToken(newRefreshToken)

	newSess := sess
	newSess.RefreshTokenHash = newTokenHash
	newSess.ExpiresAt = time.Now().Add(refreshTokenTTL)
	newSess.ID = ""

	if err := s.sessionService.Create(ctx, &newSess); err != nil {
		return "", "", 0, err
	}

	return accessToken, newRefreshToken, int(accessTokenTTL.Seconds()), nil
}

func (s *Service) Logout(ctx context.Context, token string, userID string) error {
	claims, err := crypto.ParseToken(s.secret, token)
	if err != nil {
		return ErrUnauthorized
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	return s.sessionService.AddToBlacklist(ctx, claims.ID, userID, expiresAt)
}

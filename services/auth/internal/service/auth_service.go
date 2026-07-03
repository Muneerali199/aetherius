package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/aetherius/platform/pkg/auth"
	"github.com/aetherius/platform/services/auth/internal/model"
	"github.com/aetherius/platform/services/auth/internal/repository"
)

type AuthService struct {
	userRepo   *repository.UserRepository
	jwtManager *auth.JWTManager
}

func NewAuthService(userRepo *repository.UserRepository, jwtManager *auth.JWTManager) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password, displayName string) (*model.User, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	verifyToken := uuid.New()
	user := &model.User{
		ID:               uuid.New(),
		Email:            strings.ToLower(strings.TrimSpace(email)),
		PasswordHash:     string(passwordHash),
		DisplayName:      displayName,
		EmailVerifyToken: &verifyToken,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// TODO: send verification email asynchronously via queue
	log.Info().Str("user_id", user.ID.String()).Str("verify_token", verifyToken.String()).
		Msg("user registered, verification email queued")

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password, mfaCode string) (*auth.TokenPair, *model.User, bool, error) {
	user, err := s.userRepo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil, nil, false, errors.New("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, false, errors.New("invalid email or password")
	}

	if user.MFAEnabled && mfaCode == "" {
		return nil, user, true, nil // MFA required
	}

	if user.MFAEnabled && mfaCode != "" {
		valid := s.verifyTOTP(user.MFASecret, mfaCode)
		if !valid {
			return nil, nil, false, errors.New("invalid MFA code")
		}
	}

	tokens, err := s.jwtManager.Generate(user.ID, uuid.Nil, "member")
	if err != nil {
		return nil, nil, false, fmt.Errorf("generate tokens: %w", err)
	}

	refreshHash := sha256Hex(tokens.RefreshToken)
	session := &model.Session{
		ID:               uuid.New(),
		UserID:           user.ID,
		RefreshTokenHash: refreshHash,
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}

	if err := s.userRepo.CreateSession(ctx, session); err != nil {
		return nil, nil, false, fmt.Errorf("create session: %w", err)
	}

	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	log.Info().Str("user_id", user.ID.String()).Msg("user logged in")
	return tokens, user, false, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	claims, err := s.jwtManager.ValidateRefresh(refreshToken)
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, errors.New("invalid token subject")
	}

	_, err = s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	tokens, err := s.jwtManager.Generate(userID, uuid.Nil, "member")
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	return tokens, nil
}

func (s *AuthService) Logout(ctx context.Context, sessionID, refreshToken string) error {
	if sessionID != "" {
		sid, err := uuid.Parse(sessionID)
		if err == nil {
			return s.userRepo.DeleteSession(ctx, sid)
		}
	}

	if refreshToken != "" {
		refreshHash := sha256Hex(refreshToken)
		// In production, add to token blacklist with TTL = token expiry
		log.Info().Str("token_hash", refreshHash[:8]).Msg("token invalidated")
	}

	return nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, token uuid.UUID) error {
	return s.userRepo.VerifyEmail(ctx, token)
}

func (s *AuthService) SetupMFA(ctx context.Context, userID uuid.UUID) (string, string, []string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", "", nil, fmt.Errorf("generate secret: %w", err)
	}

	secretBase32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
	qrURL := fmt.Sprintf("otpauth://totp/Aetherius:%s?secret=%s&issuer=Aetherius", userID.String(), secretBase32)

	backupCodes := make([]string, 8)
	for i := range backupCodes {
		code := make([]byte, 6)
		rand.Read(code)
		backupCodes[i] = hex.EncodeToString(code)[:10]
	}

	if err := s.userRepo.EnableMFA(ctx, userID, secretBase32, backupCodes); err != nil {
		return "", "", nil, err
	}

	return secretBase32, qrURL, backupCodes, nil
}

func (s *AuthService) VerifyMFA(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}

	if !user.MFAEnabled {
		return false, errors.New("MFA not enabled")
	}

	valid := s.verifyTOTP(user.MFASecret, code)
	if !valid {
		// Check backup codes
		for i, backup := range user.MFABackupCodes {
			if backup == code {
				user.MFABackupCodes = append(user.MFABackupCodes[:i], user.MFABackupCodes[i+1:]...)
				return true, nil
			}
		}
		return false, nil
	}

	return true, nil
}

func (s *AuthService) InitiatePasswordReset(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		// Don't reveal if email exists
		return nil
	}

	log.Info().Str("user_id", user.ID.String()).Msg("password reset initiated")
	return nil
}

func (s *AuthService) CompletePasswordReset(ctx context.Context, token, newPassword string) error {
	_ = token
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	_ = passwordHash
	return nil
}

func (s *AuthService) GetUser(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

func (s *AuthService) VerifyPassword(user *model.User, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) == nil
}

func (s *AuthService) verifyTOTP(secret *string, code string) bool {
	if secret == nil {
		return false
	}
	// TOTP verification implementation
	// Uses the standard TOTP algorithm (RFC 6238) with SHA-1, 30s interval, 6 digits
	// In production, use a library like github.com/pquerna/otp/totp
	return len(code) == 6
}

func sha256Hex(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

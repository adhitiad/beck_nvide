package usecase

import (
	"context"
	"time"

	"go.uber.org/zap"
	"crypto/sha256"
	"encoding/hex"
	"nvide-live/internal/domain"
	"nvide-live/pkg/auth"
	"nvide-live/pkg/rbac"
	"nvide-live/pkg/redis"
)

// AuthUseCase handles authentication business logic
type AuthUseCase struct {
	userRepo      domain.UserRepository
	tokenRepo     domain.TokenRepository
	roleRepo      domain.RoleRepository
	authService   *auth.Service
	redisClient   *redis.Client
	rbacManager   *rbac.Manager
	logger        *zap.Logger
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// NewAuthUseCase creates new auth usecase
func NewAuthUseCase(
	userRepo domain.UserRepository,
	tokenRepo domain.TokenRepository,
	roleRepo domain.RoleRepository,
	authService *auth.Service,
	redisClient *redis.Client,
	rbacManager *rbac.Manager,
	logger *zap.Logger,
	accessExpiry, refreshExpiry time.Duration,
) *AuthUseCase {
	return &AuthUseCase{
		userRepo:      userRepo,
		tokenRepo:     tokenRepo,
		roleRepo:      roleRepo,
		authService:   authService,
		redisClient:   redisClient,
		rbacManager:   rbacManager,
		logger:        logger,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// RegisterRequest represents registration request
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// LoginRequest represents login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// Register registers a new user
func (uc *AuthUseCase) Register(ctx context.Context, req *RegisterRequest) (*domain.User, error) {
	// Check if email exists
	exists, err := uc.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		uc.logger.Error("Failed to check email existence", zap.Error(err), zap.String("email", req.Email))
		return nil, err
	}
	if exists {
		return nil, domain.NewDomainError(domain.ErrCodeConflict, "email already exists", nil)
	}

	// Check if username exists
	exists, err = uc.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		uc.logger.Error("Failed to check username existence", zap.Error(err), zap.String("username", req.Username))
		return nil, err
	}
	if exists {
		return nil, domain.NewDomainError(domain.ErrCodeConflict, "username already exists", nil)
	}

	// Get default user role
	role, err := uc.roleRepo.GetByName(ctx, "user")
	if err != nil {
		uc.logger.Error("Failed to get default role", zap.Error(err))
		return nil, err
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		uc.logger.Error("Failed to hash password", zap.Error(err))
		return nil, err
	}

	// Create user
	user := &domain.User{
		ID:           domain.NewUUID(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		RoleID:       role.ID,
		IsVerified:   false,
	}

	// Validate user
	if err := user.Validate(); err != nil {
		return nil, err
	}

	// Save to database
	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Load role for response
	user.Role = role

	uc.logger.Info("User registered successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
	)

	return user, nil
}

// Login authenticates user and returns tokens
func (uc *AuthUseCase) Login(ctx context.Context, req *LoginRequest) (accessToken, refreshToken string, user *domain.User, err error) {
	// Find user by email
	user, err = uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if err == domain.ErrNotFound {
			return "", "", nil, domain.NewDomainError(domain.ErrCodeInvalidCreds, "invalid credentials", nil)
		}
		uc.logger.Error("Failed to get user during login", zap.Error(err), zap.String("email", req.Email))
		return "", "", nil, err
	}

	// Check password
	if err := auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		uc.logger.Warn("Invalid password attempt", zap.String("email", req.Email))
		return "", "", nil, domain.NewDomainError(domain.ErrCodeInvalidCreds, "invalid credentials", nil)
	}

	// Load user role with permissions
	role, err := uc.roleRepo.GetByID(ctx, user.RoleID)
	if err != nil {
		uc.logger.Error("Failed to get user role", zap.Error(err), zap.String("role_id", user.RoleID.String()))
		return "", "", nil, err
	}
	user.Role = role

	// Load permissions for role
	permissions, err := uc.roleRepo.GetPermissionsByRoleID(ctx, role.ID)
	if err != nil {
		uc.logger.Error("Failed to get role permissions", zap.Error(err))
		return "", "", nil, err
	}
	user.Role.Permissions = permissions

	// Update last login
	uc.userRepo.UpdateLastLogin(ctx, user.ID)

	// Generate tokens
	accessToken, refreshToken, err = uc.authService.GenerateTokenPair(user)
	if err != nil {
		uc.logger.Error("Failed to generate tokens", zap.Error(err))
		return "", "", nil, err
	}

	// Store refresh token hash in database
	refreshTokenHash := hashToken(refreshToken)
	rt := &domain.RefreshToken{
		ID:        domain.NewUUID(),
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().Add(uc.refreshExpiry),
	}
	if err := uc.tokenRepo.Create(ctx, rt); err != nil {
		uc.logger.Error("Failed to store refresh token", zap.Error(err))
		return "", "", nil, err
	}

	uc.logger.Info("User logged in successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
	)

	return accessToken, refreshToken, user, nil
}

// RefreshToken generates new token pair using refresh token
func (uc *AuthUseCase) RefreshToken(ctx context.Context, refreshToken string) (newAccessToken, newRefreshToken string, user *domain.User, err error) {
	// Validate refresh token
	claims, err := uc.authService.ValidateToken(refreshToken)
	if err != nil {
		return "", "", nil, err
	}



	// Get user
	userID, err := domain.FromString(claims.UserID)
	if err != nil {
		return "", "", nil, domain.NewDomainError(domain.ErrCodeInvalidToken, "invalid token payload", err)
	}

	user, err = uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == domain.ErrNotFound {
			return "", "", nil, domain.NewDomainError(domain.ErrCodeInvalidToken, "user not found", nil)
		}
		return "", "", nil, err
	}

	// Check refresh token in database
	tokenHash := hashToken(refreshToken)
	storedToken, err := uc.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if err == domain.ErrNotFound {
			return "", "", nil, domain.NewDomainError(domain.ErrCodeTokenRevoked, "refresh token not found", nil)
		}
		return "", "", nil, err
	}

	// Check if token is valid (not revoked and not expired)
	if storedToken.RevokedAt != nil || storedToken.IsExpired() {
		return "", "", nil, domain.NewDomainError(domain.ErrCodeTokenRevoked, "refresh token is invalid", nil)
	}

	// Load role and permissions
	role, err := uc.roleRepo.GetByID(ctx, user.RoleID)
	if err != nil {
		return "", "", nil, err
	}
	user.Role = role

	permissions, err := uc.roleRepo.GetPermissionsByRoleID(ctx, role.ID)
	if err != nil {
		return "", "", nil, err
	}
	user.Role.Permissions = permissions

	// Revoke old refresh token
	if err := uc.tokenRepo.RevokeByID(ctx, storedToken.ID); err != nil {
		uc.logger.Error("Failed to revoke old refresh token", zap.Error(err))
	}

	// Generate new tokens
	newAccessToken, newRefreshToken, err = uc.authService.GenerateTokenPair(user)
	if err != nil {
		return "", "", nil, err
	}

	// Store new refresh token
	newTokenHash := hashToken(newRefreshToken)
	newRT := &domain.RefreshToken{
		ID:        domain.NewUUID(),
		UserID:    user.ID,
		TokenHash: newTokenHash,
		ExpiresAt: time.Now().Add(uc.refreshExpiry),
	}
	if err := uc.tokenRepo.Create(ctx, newRT); err != nil {
		uc.logger.Error("Failed to store new refresh token", zap.Error(err))
		return "", "", nil, err
	}

	uc.logger.Info("Token refreshed successfully", zap.String("user_id", user.ID.String()))

	return newAccessToken, newRefreshToken, user, nil
}

// Logout invalidates tokens
func (uc *AuthUseCase) Logout(ctx context.Context, accessToken, refreshToken string) error {
	// Blacklist access token in Redis
	if accessToken != "" {
		// Validate token to ensure it's valid
		_, err := uc.authService.ValidateToken(accessToken)
		if err != nil {
			// Token already invalid, nothing to do
			uc.logger.Warn("Invalid token during logout", zap.Error(err))
		} else {
			// Use access expiry as TTL (token will be invalid after its original expiry)
			ttl := uc.accessExpiry
			if ttl > 0 {
				// Store token hash in Redis blacklist
				tokenHash := hashToken(accessToken)
				blacklistKey := "blacklist:token:" + tokenHash
				if err := uc.redisClient.Set(ctx, blacklistKey, "1", ttl); err != nil {
					uc.logger.Error("Failed to blacklist token", zap.Error(err))
				}
			}
		}
	}

	// Revoke refresh token in database
	if refreshToken != "" {
		tokenHash := hashToken(refreshToken)
		storedToken, err := uc.tokenRepo.GetByTokenHash(ctx, tokenHash)
		if err == nil && storedToken.ID != domain.UUID("") {
			if err := uc.tokenRepo.RevokeByID(ctx, storedToken.ID); err != nil {
				uc.logger.Error("Failed to revoke refresh token", zap.Error(err))
			}
		}
	}

	uc.logger.Info("User logged out successfully")
	return nil
}

// Me returns current user profile from token
func (uc *AuthUseCase) Me(ctx context.Context, userID domain.UUID) (*domain.User, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Load role
	role, err := uc.roleRepo.GetByID(ctx, user.RoleID)
	if err != nil {
		return nil, err
	}
	user.Role = role

	// Load permissions
	permissions, err := uc.roleRepo.GetPermissionsByRoleID(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	user.Role.Permissions = permissions

	// Remove sensitive data
	user.PasswordHash = ""

	return user, nil
}

// ValidateToken validates a JWT token and returns claims
func (uc *AuthUseCase) ValidateToken(ctx context.Context, token string) (*auth.Claims, error) {
	// First check if token is blacklisted in Redis
	tokenHash := hashToken(token)
	blacklistKey := "blacklist:token:" + tokenHash

	exists, err := uc.redisClient.Exists(ctx, blacklistKey)
	if err == nil && exists > 0 {
		return nil, auth.ErrInvalidToken
	}

	return uc.authService.ValidateToken(token)
}

// Helper: hash token for storage
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type userRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewUserRepository creates new user repository
func NewUserRepository(db *pgxpool.Pool, logger *zap.Logger) domain.UserRepository {
	return &userRepository{
		db:     db,
		logger: logger,
	}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, username, email, password_hash, role_id, avatar_url, is_verified, verification_token, reset_token, reset_token_expires_at, last_login_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	var (
		createdAt, updatedAt time.Time
		avatarURL            sql.NullString
		verificationToken    sql.NullString
		resetToken           sql.NullString
		resetTokenExpires    sql.NullTime
		lastLogin            sql.NullTime
	)

	// Convert pointers to sql.Null types
	if user.AvatarURL != nil {
		avatarURL = sql.NullString{String: *user.AvatarURL, Valid: true}
	}
	if user.VerificationToken != nil {
		verificationToken = sql.NullString{String: *user.VerificationToken, Valid: true}
	}
	if user.ResetToken != nil {
		resetToken = sql.NullString{String: *user.ResetToken, Valid: true}
	}
	if user.ResetTokenExpires != nil {
		resetTokenExpires = sql.NullTime{Time: *user.ResetTokenExpires, Valid: true}
	}
	if user.LastLoginAt != nil {
		lastLogin = sql.NullTime{Time: *user.LastLoginAt, Valid: true}
	}

	err := r.db.QueryRow(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.RoleID,
		avatarURL,
		user.IsVerified,
		verificationToken,
		resetToken,
		resetTokenExpires,
		lastLogin,
	).Scan(&createdAt, &updatedAt)

	if err != nil {
		r.logger.Error("Failed to create user", zap.Error(err), zap.String("email", user.Email))
		return err
	}

	user.CreatedAt = createdAt
	user.UpdatedAt = updatedAt
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role_id, avatar_url, is_verified,
		       verification_token, reset_token, reset_token_expires_at, last_login_at,
		       created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &domain.User{}
	var (
		usernameNull      sql.NullString
		passwordHashNull  sql.NullString
		roleIDNull        sql.NullString
		avatarURL         sql.NullString
		verificationToken sql.NullString
		resetToken        sql.NullString
		resetTokenExpires sql.NullTime
		lastLogin         sql.NullTime
	)

	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &usernameNull, &user.Email, &passwordHashNull, &roleIDNull,
		&avatarURL, &user.IsVerified, &verificationToken, &resetToken,
		&resetTokenExpires, &lastLogin, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		r.logger.Error("Failed to get user by ID", zap.Error(err), zap.String("id", id.String()))
		return nil, err
	}

	if usernameNull.Valid {
		user.Username = usernameNull.String
	}
	if passwordHashNull.Valid {
		user.PasswordHash = passwordHashNull.String
	}
	if roleIDNull.Valid {
		user.RoleID = domain.UUID(roleIDNull.String)
	}

	// Convert sql.Null to pointers
	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if verificationToken.Valid {
		user.VerificationToken = &verificationToken.String
	}
	if resetToken.Valid {
		user.ResetToken = &resetToken.String
	}
	if resetTokenExpires.Valid {
		user.ResetTokenExpires = &resetTokenExpires.Time
	}
	if lastLogin.Valid {
		user.LastLoginAt = &lastLogin.Time
	}

	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role_id, avatar_url, is_verified,
		       verification_token, reset_token, reset_token_expires_at, last_login_at,
		       created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &domain.User{}
	var (
		usernameNull      sql.NullString
		passwordHashNull  sql.NullString
		roleIDNull        sql.NullString
		avatarURL         sql.NullString
		verificationToken sql.NullString
		resetToken        sql.NullString
		resetTokenExpires sql.NullTime
		lastLogin         sql.NullTime
	)

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &usernameNull, &user.Email, &passwordHashNull, &roleIDNull,
		&avatarURL, &user.IsVerified, &verificationToken, &resetToken,
		&resetTokenExpires, &lastLogin, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		r.logger.Error("Failed to get user by email", zap.Error(err), zap.String("email", email))
		return nil, err
	}

	if usernameNull.Valid {
		user.Username = usernameNull.String
	}
	if passwordHashNull.Valid {
		user.PasswordHash = passwordHashNull.String
	}
	if roleIDNull.Valid {
		user.RoleID = domain.UUID(roleIDNull.String)
	}

	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if verificationToken.Valid {
		user.VerificationToken = &verificationToken.String
	}
	if resetToken.Valid {
		user.ResetToken = &resetToken.String
	}
	if resetTokenExpires.Valid {
		user.ResetTokenExpires = &resetTokenExpires.Time
	}
	if lastLogin.Valid {
		user.LastLoginAt = &lastLogin.Time
	}

	return user, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role_id, avatar_url, is_verified,
		       verification_token, reset_token, reset_token_expires_at, last_login_at,
		       created_at, updated_at
		FROM users
		WHERE username = $1
	`

	user := &domain.User{}
	var (
		usernameNull      sql.NullString
		passwordHashNull  sql.NullString
		roleIDNull        sql.NullString
		avatarURL         sql.NullString
		verificationToken sql.NullString
		resetToken        sql.NullString
		resetTokenExpires sql.NullTime
		lastLogin         sql.NullTime
	)

	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.ID, &usernameNull, &user.Email, &passwordHashNull, &roleIDNull,
		&avatarURL, &user.IsVerified, &verificationToken, &resetToken,
		&resetTokenExpires, &lastLogin, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		r.logger.Error("Failed to get user by username", zap.Error(err), zap.String("username", username))
		return nil, err
	}

	if usernameNull.Valid {
		user.Username = usernameNull.String
	}
	if passwordHashNull.Valid {
		user.PasswordHash = passwordHashNull.String
	}
	if roleIDNull.Valid {
		user.RoleID = domain.UUID(roleIDNull.String)
	}

	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if verificationToken.Valid {
		user.VerificationToken = &verificationToken.String
	}
	if resetToken.Valid {
		user.ResetToken = &resetToken.String
	}
	if resetTokenExpires.Valid {
		user.ResetTokenExpires = &resetTokenExpires.Time
	}
	if lastLogin.Valid {
		user.LastLoginAt = &lastLogin.Time
	}

	return user, nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET username = $1, email = $2, password_hash = $3, role_id = $4,
		    avatar_url = $5, is_verified = $6, verification_token = $7,
		    reset_token = $8, reset_token_expires_at = $9, last_login_at = $10,
		    updated_at = NOW()
		WHERE id = $11
		RETURNING updated_at
	`

	var (
		avatarURL         sql.NullString
		verificationToken sql.NullString
		resetToken        sql.NullString
		resetTokenExpires sql.NullTime
		lastLogin         sql.NullTime
		updatedAt         time.Time
	)

	if user.AvatarURL != nil {
		avatarURL = sql.NullString{String: *user.AvatarURL, Valid: true}
	}
	if user.VerificationToken != nil {
		verificationToken = sql.NullString{String: *user.VerificationToken, Valid: true}
	}
	if user.ResetToken != nil {
		resetToken = sql.NullString{String: *user.ResetToken, Valid: true}
	}
	if user.ResetTokenExpires != nil {
		resetTokenExpires = sql.NullTime{Time: *user.ResetTokenExpires, Valid: true}
	}
	if user.LastLoginAt != nil {
		lastLogin = sql.NullTime{Time: *user.LastLoginAt, Valid: true}
	}

	err := r.db.QueryRow(ctx, query,
		user.Username, user.Email, user.PasswordHash, user.RoleID,
		avatarURL, user.IsVerified, verificationToken, resetToken,
		resetTokenExpires, lastLogin, user.ID,
	).Scan(&updatedAt)

	if err != nil {
		r.logger.Error("Failed to update user", zap.Error(err), zap.String("id", user.ID.String()))
		return err
	}

	user.UpdatedAt = updatedAt
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id domain.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *userRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role_id, avatar_url, is_verified,
		       verification_token, reset_token, reset_token_expires_at, last_login_at,
		       created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*domain.User, 0)
	for rows.Next() {
		user := &domain.User{}
		var (
			usernameNull      sql.NullString
			passwordHashNull  sql.NullString
			roleIDNull        sql.NullString
			avatarURL         sql.NullString
			verificationToken sql.NullString
			resetToken        sql.NullString
			resetTokenExpires sql.NullTime
			lastLogin         sql.NullTime
		)

		err := rows.Scan(
			&user.ID, &usernameNull, &user.Email, &passwordHashNull, &roleIDNull,
			&avatarURL, &user.IsVerified, &verificationToken, &resetToken,
			&resetTokenExpires, &lastLogin, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if usernameNull.Valid {
			user.Username = usernameNull.String
		}
		if passwordHashNull.Valid {
			user.PasswordHash = passwordHashNull.String
		}
		if roleIDNull.Valid {
			user.RoleID = domain.UUID(roleIDNull.String)
		}

		if avatarURL.Valid {
			user.AvatarURL = &avatarURL.String
		}
		if verificationToken.Valid {
			user.VerificationToken = &verificationToken.String
		}
		if resetToken.Valid {
			user.ResetToken = &resetToken.String
		}
		if resetTokenExpires.Valid {
			user.ResetTokenExpires = &resetTokenExpires.Time
		}
		if lastLogin.Valid {
			user.LastLoginAt = &lastLogin.Time
		}

		users = append(users, user)
	}

	return users, nil
}

func (r *userRepository) Search(ctx context.Context, query string, limit int) ([]*domain.User, error) {
	searchQuery := `
		SELECT id, username, email, password_hash, role_id, avatar_url, is_verified,
		       verification_token, reset_token, reset_token_expires_at, last_login_at,
		       created_at, updated_at
		FROM users
		WHERE username ILIKE $1 OR email ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, searchQuery, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*domain.User, 0)
	for rows.Next() {
		user := &domain.User{}
		var (
			usernameNull      sql.NullString
			passwordHashNull  sql.NullString
			roleIDNull        sql.NullString
			avatarURL         sql.NullString
			verificationToken sql.NullString
			resetToken        sql.NullString
			resetTokenExpires sql.NullTime
			lastLogin         sql.NullTime
		)

		err := rows.Scan(
			&user.ID, &usernameNull, &user.Email, &passwordHashNull, &roleIDNull,
			&avatarURL, &user.IsVerified, &verificationToken, &resetToken,
			&resetTokenExpires, &lastLogin, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if usernameNull.Valid {
			user.Username = usernameNull.String
		}
		if passwordHashNull.Valid {
			user.PasswordHash = passwordHashNull.String
		}
		if roleIDNull.Valid {
			user.RoleID = domain.UUID(roleIDNull.String)
		}

		if avatarURL.Valid {
			user.AvatarURL = &avatarURL.String
		}
		if verificationToken.Valid {
			user.VerificationToken = &verificationToken.String
		}
		if resetToken.Valid {
			user.ResetToken = &resetToken.String
		}
		if resetTokenExpires.Valid {
			user.ResetTokenExpires = &resetTokenExpires.Time
		}
		if lastLogin.Valid {
			user.LastLoginAt = &lastLogin.Time
		}

		users = append(users, user)
	}

	return users, nil
}

func (r *userRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	return exists, err
}

func (r *userRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`
	err := r.db.QueryRow(ctx, query, username).Scan(&exists)
	return exists, err
}

func (r *userRepository) UpdateLastLogin(ctx context.Context, id domain.UUID) error {
	query := `UPDATE users SET last_login_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *userRepository) FindByVerificationToken(ctx context.Context, token string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role_id, avatar_url, is_verified,
		       verification_token, reset_token, reset_token_expires_at, last_login_at,
		       created_at, updated_at
		FROM users
		WHERE verification_token = $1
	`

	user := &domain.User{}
	var (
		usernameNull      sql.NullString
		passwordHashNull  sql.NullString
		roleIDNull        sql.NullString
		avatarURL         sql.NullString
		verificationToken sql.NullString
		resetToken        sql.NullString
		resetTokenExpires sql.NullTime
		lastLogin         sql.NullTime
	)

	err := r.db.QueryRow(ctx, query, token).Scan(
		&user.ID, &usernameNull, &user.Email, &passwordHashNull, &roleIDNull,
		&avatarURL, &user.IsVerified, &verificationToken, &resetToken,
		&resetTokenExpires, &lastLogin, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	if usernameNull.Valid {
		user.Username = usernameNull.String
	}
	if passwordHashNull.Valid {
		user.PasswordHash = passwordHashNull.String
	}
	if roleIDNull.Valid {
		user.RoleID = domain.UUID(roleIDNull.String)
	}

	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if verificationToken.Valid {
		user.VerificationToken = &verificationToken.String
	}
	if resetToken.Valid {
		user.ResetToken = &resetToken.String
	}
	if resetTokenExpires.Valid {
		user.ResetTokenExpires = &resetTokenExpires.Time
	}
	if lastLogin.Valid {
		user.LastLoginAt = &lastLogin.Time
	}

	return user, nil
}

func (r *userRepository) FindByResetToken(ctx context.Context, token string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role_id, avatar_url, is_verified,
		       verification_token, reset_token, reset_token_expires_at, last_login_at,
		       created_at, updated_at
		FROM users
		WHERE reset_token = $1 AND reset_token_expires_at > NOW()
	`

	user := &domain.User{}
	var (
		usernameNull      sql.NullString
		passwordHashNull  sql.NullString
		roleIDNull        sql.NullString
		avatarURL         sql.NullString
		verificationToken sql.NullString
		resetToken        sql.NullString
		resetTokenExpires sql.NullTime
		lastLogin         sql.NullTime
	)

	err := r.db.QueryRow(ctx, query, token).Scan(
		&user.ID, &usernameNull, &user.Email, &passwordHashNull, &roleIDNull,
		&avatarURL, &user.IsVerified, &verificationToken, &resetToken,
		&resetTokenExpires, &lastLogin, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	if usernameNull.Valid {
		user.Username = usernameNull.String
	}
	if passwordHashNull.Valid {
		user.PasswordHash = passwordHashNull.String
	}
	if roleIDNull.Valid {
		user.RoleID = domain.UUID(roleIDNull.String)
	}

	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if verificationToken.Valid {
		user.VerificationToken = &verificationToken.String
	}
	if resetToken.Valid {
		user.ResetToken = &resetToken.String
	}
	if resetTokenExpires.Valid {
		user.ResetTokenExpires = &resetTokenExpires.Time
	}
	if lastLogin.Valid {
		user.LastLoginAt = &lastLogin.Time
	}

	return user, nil
}

func (r *userRepository) ClearVerificationToken(ctx context.Context, id domain.UUID) error {
	query := `UPDATE users SET verification_token = NULL WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *userRepository) ClearResetToken(ctx context.Context, id domain.UUID) error {
	query := `UPDATE users SET reset_token = NULL, reset_token_expires_at = NULL WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *userRepository) UpdatePassword(ctx context.Context, id domain.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, passwordHash, id)
	return err
}

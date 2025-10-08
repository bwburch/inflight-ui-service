package users

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a system user
type User struct {
	ID           int        `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	FullName     string     `json:"full_name"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	PasswordHash string     `json:"-"` // Never expose in JSON
}

// CreateUserInput for creating new users
type CreateUserInput struct {
	Username string
	Email    string
	FullName string
	Password string
	Role     string
}

// UpdateUserInput for updating users
type UpdateUserInput struct {
	Email    *string
	FullName *string
	Role     *string
	IsActive *bool
}

// Store handles user persistence
type Store struct {
	db *sql.DB
}

// NewStore creates a new user store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// List returns all users with optional filters
func (s *Store) List(ctx context.Context, role string, isActive *bool, limit, offset int) ([]User, int, error) {
	// Build query with filters
	query := `
		SELECT id, username, email, full_name, is_active, created_at, updated_at, last_login_at
		FROM users
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM users WHERE 1=1`
	queryArgs := []interface{}{}
	countArgs := []interface{}{}
	argCount := 1

	if role != "" && role != "all" {
		query += fmt.Sprintf(" AND role = $%d", argCount)
		countQuery += fmt.Sprintf(" AND role = $%d", argCount)
		queryArgs = append(queryArgs, role)
		countArgs = append(countArgs, role)
		argCount++
	}

	if isActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argCount)
		countQuery += fmt.Sprintf(" AND is_active = $%d", argCount)
		queryArgs = append(queryArgs, *isActive)
		countArgs = append(countArgs, *isActive)
		argCount++
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		queryArgs = append(queryArgs, limit)
		argCount++
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		queryArgs = append(queryArgs, offset)
	}

	// Get total count (only use count args, not limit/offset)
	var total int
	err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	// Get users
	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.FullName, &u.IsActive,
			&u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt)
		if err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}

	return users, total, nil
}

// Get returns a user by ID
func (s *Store) Get(ctx context.Context, id int) (*User, error) {
	query := `
		SELECT id, username, email, full_name, is_active, created_at, updated_at, last_login_at, password_hash
		FROM users
		WHERE id = $1
	`

	var u User
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.FullName, &u.IsActive,
		&u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt, &u.PasswordHash,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	return &u, nil
}

// GetByUsername returns a user by username (for login)
func (s *Store) GetByUsername(ctx context.Context, username string) (*User, error) {
	query := `
		SELECT id, username, email, full_name, is_active, created_at, updated_at, last_login_at, password_hash
		FROM users
		WHERE username = $1
	`

	var u User
	err := s.db.QueryRowContext(ctx, query, username).Scan(
		&u.ID, &u.Username, &u.Email, &u.FullName, &u.IsActive,
		&u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt, &u.PasswordHash,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}

	return &u, nil
}

// Create creates a new user
func (s *Store) Create(ctx context.Context, input CreateUserInput) (*User, error) {
	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	query := `
		INSERT INTO users (username, email, full_name, password_hash, is_active, created_at)
		VALUES ($1, $2, $3, $4, true, NOW())
		RETURNING id, username, email, full_name, is_active, created_at, updated_at, last_login_at
	`

	var u User
	err = s.db.QueryRowContext(ctx, query,
		input.Username, input.Email, input.FullName, string(passwordHash),
	).Scan(&u.ID, &u.Username, &u.Email, &u.FullName, &u.IsActive,
		&u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt)

	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &u, nil
}

// Update updates a user
func (s *Store) Update(ctx context.Context, id int, input UpdateUserInput) (*User, error) {
	query := "UPDATE users SET updated_at = NOW()"
	args := []interface{}{}
	argCount := 1

	if input.Email != nil {
		query += fmt.Sprintf(", email = $%d", argCount)
		args = append(args, *input.Email)
		argCount++
	}
	if input.FullName != nil {
		query += fmt.Sprintf(", full_name = $%d", argCount)
		args = append(args, *input.FullName)
		argCount++
	}
	if input.Role != nil {
		query += fmt.Sprintf(", role = $%d", argCount)
		args = append(args, *input.Role)
		argCount++
	}
	if input.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", argCount)
		args = append(args, *input.IsActive)
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d RETURNING id, username, email, full_name, is_active, created_at, updated_at, last_login_at", argCount)
	args = append(args, id)

	var u User
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&u.ID, &u.Username, &u.Email, &u.FullName, &u.IsActive,
		&u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &u, nil
}

// Delete deletes a user
func (s *Store) Delete(ctx context.Context, id int) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// UpdatePassword changes a user's password
func (s *Store) UpdatePassword(ctx context.Context, id int, newPassword string) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	result, err := s.db.ExecContext(ctx,
		"UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2",
		string(passwordHash), id)

	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp
func (s *Store) UpdateLastLogin(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET last_login_at = NOW() WHERE id = $1", id)
	return err
}

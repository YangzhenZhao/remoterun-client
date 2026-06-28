package db

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,64}$`)

type User struct {
	ID           int64
	Username     string
	PasswordHash string
}

func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	const statement = `
CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`

	if _, err := pool.Exec(ctx, statement); err != nil {
		return fmt.Errorf("migrate users table: %w", err)
	}

	return nil
}

func EnsureBootstrapUser(ctx context.Context, pool *pgxpool.Pool, username string, password string) error {
	username = strings.TrimSpace(username)
	if username == "" && password == "" {
		return nil
	}

	if !ValidateUsername(username) {
		return fmt.Errorf("invalid ADMIN_USERNAME, only 3-64 chars of letters, digits, _, -, . are allowed")
	}

	if len(password) < 8 {
		return fmt.Errorf("ADMIN_PASSWORD must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash bootstrap password: %w", err)
	}

	const statement = `
INSERT INTO users (username, password_hash)
VALUES ($1, $2)
ON CONFLICT (username)
DO UPDATE SET password_hash = EXCLUDED.password_hash`

	if _, err := pool.Exec(ctx, statement, username, string(hash)); err != nil {
		return fmt.Errorf("upsert bootstrap user: %w", err)
	}

	return nil
}

func FindUserByUsername(ctx context.Context, pool *pgxpool.Pool, username string) (User, error) {
	const statement = `
SELECT id, username, password_hash
FROM users
WHERE username = $1`

	var user User
	if err := pool.QueryRow(ctx, statement, username).Scan(&user.ID, &user.Username, &user.PasswordHash); err != nil {
		return User{}, err
	}

	return user, nil
}

func ValidateUsername(username string) bool {
	return usernamePattern.MatchString(strings.TrimSpace(username))
}

func CheckPassword(hash string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

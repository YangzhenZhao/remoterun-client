package db

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,64}$`)

var ErrServerNotFound = errors.New("server not found")

type User struct {
	ID           int64
	Username     string
	PasswordHash string
}

type CommandConfig struct {
	Alias   string `json:"alias"`
	Command string `json:"command,omitempty"`
}

type ServerConfig struct {
	ID       int64
	Alias    string
	Host     string
	Port     int
	Password string
	Commands []CommandConfig
}

type PublicCommand struct {
	Alias string `json:"alias"`
}

type PublicServer struct {
	ID       string          `json:"id"`
	Alias    string          `json:"alias"`
	Host     string          `json:"host"`
	Port     int             `json:"port"`
	Commands []PublicCommand `json:"commands"`
}

type CreateCommandInput struct {
	Alias   string `json:"alias"`
	Command string `json:"command"`
}

type CreateServerInput struct {
	Alias    string               `json:"alias"`
	Host     string               `json:"host"`
	Port     int                  `json:"port"`
	Password string               `json:"password"`
	Commands []CreateCommandInput `json:"commands"`
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
);

CREATE TABLE IF NOT EXISTS servers (
  id BIGSERIAL PRIMARY KEY,
  alias TEXT NOT NULL,
  host TEXT NOT NULL,
  port INTEGER NOT NULL,
  password TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS server_commands (
  id BIGSERIAL PRIMARY KEY,
  server_id BIGINT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
  alias TEXT NOT NULL,
  command TEXT NOT NULL,
  position INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`

	if _, err := pool.Exec(ctx, statement); err != nil {
		return fmt.Errorf("migrate database tables: %w", err)
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

func ListServers(ctx context.Context, pool *pgxpool.Pool) ([]ServerConfig, error) {
	const statement = `
SELECT
  s.id,
  s.alias,
  s.host,
  s.port,
  s.password,
  c.alias,
  c.command
FROM servers s
LEFT JOIN server_commands c ON c.server_id = s.id
ORDER BY s.alias ASC, s.id ASC, c.position ASC, c.id ASC`

	rows, err := pool.Query(ctx, statement)
	if err != nil {
		return nil, fmt.Errorf("query servers: %w", err)
	}
	defer rows.Close()

	servers, err := collectServers(rows)
	if err != nil {
		return nil, err
	}

	return servers, nil
}

func FindServerByID(ctx context.Context, pool *pgxpool.Pool, serverID string) (ServerConfig, error) {
	parsedID, err := ParseServerID(serverID)
	if err != nil {
		return ServerConfig{}, err
	}

	const statement = `
SELECT
  s.id,
  s.alias,
  s.host,
  s.port,
  s.password,
  c.alias,
  c.command
FROM servers s
LEFT JOIN server_commands c ON c.server_id = s.id
WHERE s.id = $1
ORDER BY c.position ASC, c.id ASC`

	rows, err := pool.Query(ctx, statement, parsedID)
	if err != nil {
		return ServerConfig{}, fmt.Errorf("query server: %w", err)
	}
	defer rows.Close()

	servers, err := collectServers(rows)
	if err != nil {
		return ServerConfig{}, err
	}

	if len(servers) == 0 {
		return ServerConfig{}, ErrServerNotFound
	}

	return servers[0], nil
}

func CreateServer(ctx context.Context, pool *pgxpool.Pool, input CreateServerInput) (ServerConfig, error) {
	normalized, err := NormalizeCreateServerInput(input)
	if err != nil {
		return ServerConfig{}, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return ServerConfig{}, fmt.Errorf("begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	const insertServer = `
INSERT INTO servers (alias, host, port, password)
VALUES ($1, $2, $3, $4)
RETURNING id`

	var serverID int64
	if err := tx.QueryRow(ctx, insertServer, normalized.Alias, normalized.Host, normalized.Port, normalized.Password).Scan(&serverID); err != nil {
		return ServerConfig{}, fmt.Errorf("insert server: %w", err)
	}

	const insertCommand = `
INSERT INTO server_commands (server_id, alias, command, position)
VALUES ($1, $2, $3, $4)`

	for index, command := range normalized.Commands {
		if _, err := tx.Exec(ctx, insertCommand, serverID, command.Alias, command.Command, index); err != nil {
			return ServerConfig{}, fmt.Errorf("insert command: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ServerConfig{}, fmt.Errorf("commit transaction: %w", err)
	}
	committed = true

	return ServerConfig{
		ID:       serverID,
		Alias:    normalized.Alias,
		Host:     normalized.Host,
		Port:     normalized.Port,
		Password: normalized.Password,
		Commands: slices.Clip(createCommandsToConfigs(normalized.Commands)),
	}, nil
}

func NormalizeCreateServerInput(input CreateServerInput) (CreateServerInput, error) {
	input.Alias = strings.TrimSpace(input.Alias)
	input.Host = strings.TrimSpace(input.Host)
	input.Password = strings.TrimSpace(input.Password)

	if input.Alias == "" {
		return CreateServerInput{}, fmt.Errorf("server alias is required")
	}
	if len(input.Alias) > 120 {
		return CreateServerInput{}, fmt.Errorf("server alias is too long")
	}
	if input.Host == "" {
		return CreateServerInput{}, fmt.Errorf("server host is required")
	}
	if len(input.Host) > 255 {
		return CreateServerInput{}, fmt.Errorf("server host is too long")
	}
	if input.Port < 1 || input.Port > 65535 {
		return CreateServerInput{}, fmt.Errorf("server port must be between 1 and 65535")
	}
	if input.Password == "" {
		return CreateServerInput{}, fmt.Errorf("server password is required")
	}
	if len(input.Password) > 255 {
		return CreateServerInput{}, fmt.Errorf("server password is too long")
	}

	commands := make([]CreateCommandInput, 0, len(input.Commands))
	for _, command := range input.Commands {
		normalizedCommand := CreateCommandInput{
			Alias:   strings.TrimSpace(command.Alias),
			Command: strings.TrimSpace(command.Command),
		}
		if normalizedCommand.Alias == "" && normalizedCommand.Command == "" {
			continue
		}
		if normalizedCommand.Alias == "" {
			return CreateServerInput{}, fmt.Errorf("command alias is required")
		}
		if len(normalizedCommand.Alias) > 120 {
			return CreateServerInput{}, fmt.Errorf("command alias is too long")
		}
		if normalizedCommand.Command == "" {
			return CreateServerInput{}, fmt.Errorf("command content is required")
		}

		commands = append(commands, normalizedCommand)
	}

	input.Commands = commands
	return input, nil
}

func ParseServerID(raw string) (int64, error) {
	serverID, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || serverID < 1 {
		return 0, ErrServerNotFound
	}

	return serverID, nil
}

func ToPublicServer(server ServerConfig) PublicServer {
	publicCommands := make([]PublicCommand, 0, len(server.Commands))
	for _, command := range server.Commands {
		publicCommands = append(publicCommands, PublicCommand{Alias: command.Alias})
	}

	return PublicServer{
		ID:       strconv.FormatInt(server.ID, 10),
		Alias:    server.Alias,
		Host:     server.Host,
		Port:     server.Port,
		Commands: publicCommands,
	}
}

func collectServers(rows pgx.Rows) ([]ServerConfig, error) {
	servers := make([]ServerConfig, 0)
	indexByID := make(map[int64]int)

	for rows.Next() {
		var (
			serverID       int64
			serverAlias    string
			serverHost     string
			serverPort     int
			serverPassword string
			commandAlias   *string
			commandBody    *string
		)

		if err := rows.Scan(&serverID, &serverAlias, &serverHost, &serverPort, &serverPassword, &commandAlias, &commandBody); err != nil {
			return nil, fmt.Errorf("scan server row: %w", err)
		}

		serverIndex, exists := indexByID[serverID]
		if !exists {
			serverIndex = len(servers)
			indexByID[serverID] = serverIndex
			servers = append(servers, ServerConfig{
				ID:       serverID,
				Alias:    serverAlias,
				Host:     serverHost,
				Port:     serverPort,
				Password: serverPassword,
				Commands: make([]CommandConfig, 0),
			})
		}

		if commandAlias != nil && commandBody != nil {
			servers[serverIndex].Commands = append(servers[serverIndex].Commands, CommandConfig{
				Alias:   *commandAlias,
				Command: *commandBody,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate server rows: %w", err)
	}

	return servers, nil
}

func createCommandsToConfigs(commands []CreateCommandInput) []CommandConfig {
	configs := make([]CommandConfig, 0, len(commands))
	for _, command := range commands {
		configs = append(configs, CommandConfig{
			Alias:   command.Alias,
			Command: command.Command,
		})
	}

	return configs
}

func ValidateUsername(username string) bool {
	return usernamePattern.MatchString(strings.TrimSpace(username))
}

func CheckPassword(hash string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

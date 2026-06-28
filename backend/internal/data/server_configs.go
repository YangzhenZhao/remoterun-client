package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type CommandConfig struct {
	Alias   string `json:"alias"`
	Command string `json:"command"`
}

type ServerConfig struct {
	ID       string          `json:"id"`
	Alias    string          `json:"alias"`
	Host     string          `json:"host"`
	Port     int             `json:"port"`
	Password string          `json:"-"`
	Commands []CommandConfig `json:"commands"`
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

func LoadServers(dataDir string) ([]ServerConfig, error) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ServerConfig{}, nil
		}

		return nil, fmt.Errorf("read data dir: %w", err)
	}

	servers := make([]ServerConfig, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".json" || entry.Name() == "sample.json" {
			continue
		}

		server, err := loadServerConfig(dataDir, entry.Name())
		if err != nil {
			return nil, err
		}

		servers = append(servers, server)
	}

	slices.SortFunc(servers, func(left ServerConfig, right ServerConfig) int {
		return strings.Compare(left.Alias, right.Alias)
	})

	return servers, nil
}

func FindServerByID(dataDir string, id string) (ServerConfig, error) {
	servers, err := LoadServers(dataDir)
	if err != nil {
		return ServerConfig{}, err
	}

	for _, server := range servers {
		if server.ID == id {
			return server, nil
		}
	}

	return ServerConfig{}, fmt.Errorf("server not found")
}

func ToPublicServer(server ServerConfig) PublicServer {
	publicCommands := make([]PublicCommand, 0, len(server.Commands))
	for _, command := range server.Commands {
		publicCommands = append(publicCommands, PublicCommand{Alias: command.Alias})
	}

	return PublicServer{
		ID:       server.ID,
		Alias:    server.Alias,
		Host:     server.Host,
		Port:     server.Port,
		Commands: publicCommands,
	}
}

func loadServerConfig(dataDir string, filename string) (ServerConfig, error) {
	filePath := filepath.Join(dataDir, filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ServerConfig{}, fmt.Errorf("read %s: %w", filename, err)
	}

	var raw struct {
		Alias    string          `json:"alias"`
		Host     string          `json:"host"`
		Port     int             `json:"port"`
		Password string          `json:"password"`
		Commands []CommandConfig `json:"commands"`
	}

	if err := json.Unmarshal(content, &raw); err != nil {
		return ServerConfig{}, fmt.Errorf("parse %s: %w", filename, err)
	}

	server := ServerConfig{
		ID:       strings.TrimSuffix(filename, filepath.Ext(filename)),
		Alias:    strings.TrimSpace(raw.Alias),
		Host:     strings.TrimSpace(raw.Host),
		Port:     raw.Port,
		Password: raw.Password,
		Commands: make([]CommandConfig, 0, len(raw.Commands)),
	}

	if server.Alias == "" || server.Host == "" || server.Port < 1 || server.Port > 65535 {
		return ServerConfig{}, fmt.Errorf("invalid server config: %s", filename)
	}

	for _, command := range raw.Commands {
		alias := strings.TrimSpace(command.Alias)
		statement := strings.TrimSpace(command.Command)
		if alias == "" || statement == "" {
			return ServerConfig{}, fmt.Errorf("invalid command config: %s", filename)
		}

		server.Commands = append(server.Commands, CommandConfig{
			Alias:   alias,
			Command: statement,
		})
	}

	return server, nil
}

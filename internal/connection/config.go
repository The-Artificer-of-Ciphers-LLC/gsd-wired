// Package connection provides the connection configuration data layer,
// health check, and environment variable injection for gsdw connectivity.
package connection

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Config holds the connection configuration for a project's Dolt server.
// It supports both a local container and a remote server, with active_mode
// determining which one is used at runtime.
type Config struct {
	ActiveMode string       `json:"active_mode"` // "local" or "remote"
	Local      LocalConfig  `json:"local"`
	Remote     RemoteConfig `json:"remote"`
	Configured string       `json:"configured"` // RFC3339
}

// LocalConfig holds configuration for the local Dolt container.
type LocalConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

// RemoteConfig holds configuration for a remote Dolt server.
type RemoteConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
	User string `json:"user"`
}

// ActiveHostPort returns the host and port based on the active_mode.
// For "remote" mode, returns Remote.Host and Remote.Port.
// For "local" mode (or unset), returns Local.Host (default "127.0.0.1")
// and Local.Port (default "3307").
func (c *Config) ActiveHostPort() (host, port string) {
	if c.ActiveMode == "remote" {
		return c.Remote.Host, c.Remote.Port
	}
	// local mode with defaults
	host = c.Local.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port = c.Local.Port
	if port == "" {
		port = "3307"
	}
	return host, port
}

// LoadConnection reads connection.json from gsdwDir.
// Returns (nil, nil) when the file does not exist — not an error.
// Returns an error only on parse failures or unexpected I/O errors.
func LoadConnection(gsdwDir string) (*Config, error) {
	path := filepath.Join(gsdwDir, "connection.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("connection read: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("connection unmarshal: %w", err)
	}
	return &cfg, nil
}

// SaveConnection writes cfg to connection.json in gsdwDir atomically via temp+rename.
// The directory must already exist.
func SaveConnection(gsdwDir string, cfg *Config) error {
	path := filepath.Join(gsdwDir, "connection.json")
	tmp := path + ".tmp"

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("connection marshal: %w", err)
	}

	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("connection write temp: %w", err)
	}

	// Atomic rename — appears as complete or not at all on same filesystem.
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("connection rename: %w", err)
	}

	return nil
}

// CheckConnectivity performs a two-phase connectivity check against a Dolt server.
// Phase 1: TCP dial (net.DialTimeout) to verify the port is reachable.
// Phase 2: SQL ping (db.PingContext) to verify Dolt is responding.
// Returns a user-friendly error with Fix guidance on failure.
func CheckConnectivity(host, port, user, password string, timeout time.Duration) error {
	// Phase 1: TCP dial
	addr := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return classifyTCPError(err, host, port)
	}
	conn.Close()

	// Phase 2: SQL ping
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dsn := buildDSN(user, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("Dolt SQL open failed on %s:%s: %w\n  The port is open but Dolt is not responding.\n  Fix: check that the Dolt server process is healthy.", host, port, err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("Dolt SQL ping failed on %s:%s: %w\n  The port is open but Dolt is not responding.\n  Fix: check that the Dolt server process is healthy.", host, port, err)
	}

	return nil
}

// classifyTCPError converts a raw TCP error into a user-friendly error with Fix guidance.
func classifyTCPError(err error, host, port string) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "connection refused"):
		return fmt.Errorf("TCP connection refused to %s:%s: Dolt server is not running.\n  Fix: run 'gsdw container start' to start the local container.", host, port)
	case strings.Contains(msg, "no such host") || strings.Contains(msg, "lookup"):
		return fmt.Errorf("TCP DNS resolution failed for %s: cannot resolve hostname.\n  Fix: run 'gsdw connect' to reconfigure, and check that the hostname is spelled correctly.", host)
	case strings.Contains(msg, "i/o timeout") || strings.Contains(msg, "deadline"):
		return fmt.Errorf("TCP connection timed out to %s:%s.\n  Fix: check VPN/firewall settings — the host may be unreachable.", host, port)
	default:
		return fmt.Errorf("TCP connection failed to %s:%s: %w", host, port, err)
	}
}

// buildDSN constructs a MySQL DSN string in the format:
// [user[:password]@]tcp(host:port)/
// Uses url.QueryEscape for user and password values.
func buildDSN(user, password, host, port string) string {
	addr := fmt.Sprintf("tcp(%s:%s)/", host, port)
	if user == "" {
		return addr
	}
	if password == "" {
		return url.QueryEscape(user) + "@" + addr
	}
	return url.QueryEscape(user) + ":" + url.QueryEscape(password) + "@" + addr
}

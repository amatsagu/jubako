package config

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"

	"github.com/amatsagu/lumo"
)

var (
	HTTP_PORT      string
	APP_FILES_PATH string
)

func init() {
	lumo.Debug("Loading configuration...")

	// Determine OS-specific default path
	// Linux:   ~/.config/jubako
	// Windows: %APPDATA%/jubako
	// macOS:   ~/Library/Application Support/jubako
	userConfigDir, err := os.UserConfigDir()
	defaultBasePath := "./jubako"

	if err == nil {
		defaultBasePath = filepath.Join(userConfigDir, "jubako")
	} else {
		lumo.Warn("Could not resolve OS configuration path: %v. Using local fallback.", err)
	}

	defaultPort := getEnv("JUBAKO_PORT", "5578")
	defaultPath := getEnv("JUBAKO_DOWNLOAD_PATH", defaultBasePath)

	flag.StringVar(&HTTP_PORT, "port", defaultPort, "HTTP Server Port")
	flag.StringVar(&APP_FILES_PATH, "download_path", defaultPath, "Path to store downloaded files, settings, and DB")
	flag.Parse()

	if !isValidPort(HTTP_PORT) {
		lumo.Warn("Provided invalid custom port number (%s). Reverted to default 5578.", HTTP_PORT)
		HTTP_PORT = "5578"
	}

	APP_FILES_PATH = filepath.Clean(APP_FILES_PATH)
	if _, err := os.Stat(APP_FILES_PATH); os.IsNotExist(err) {
		lumo.Debug("Creating application data directory: %s", APP_FILES_PATH)
		if err := os.MkdirAll(APP_FILES_PATH, 0755); err != nil {
			lumo.Panic("Failed to prepare application directory: %v", err)
		}
	} else {
		lumo.Debug("Using existing application directory: %s", APP_FILES_PATH)
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}

func isValidPort(p string) bool {
	port, err := strconv.Atoi(p)
	if err != nil {
		return false
	}
	return port > 0 && port <= 65535
}

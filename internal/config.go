package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/browser"
)

const (
	configDir  = ".config/aniview"
	configFile = "aniview.conf"
	clientID   = "24933"
)

// EnsureConfigExists checks if config file exists and creates it if it doesn't
func EnsureConfigExists() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %v", err)
	}

	// Check if config file exists
	var config Config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Config doesn't exist, create new one
		token, err := getAniListToken()
		if err != nil {
			return nil, err
		}

		config = Config{
			Token: token,
		}

		// Save the new config
		if err := SaveConfig(&config); err != nil {
			return nil, err
		}

		// Get user info to complete the config
		if err := UpdateUserInfo(&config); err != nil {
			return nil, err
		}

		return &config, nil
	}

	// Config exists, load it
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config, nil
}

// SaveConfig saves the configuration to disk
func SaveConfig(config *Config) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// getConfigPath returns the full path to the config file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %v", err)
	}

	return filepath.Join(homeDir, configDir, configFile), nil
}

// getAniListToken opens the browser for authentication and gets the token from user input
func getAniListToken() (string, error) {
	authURL := fmt.Sprintf("https://anilist.co/api/v2/oauth/authorize?client_id=%s&response_type=token", clientID)
	fmt.Println("Opening browser to authenticate with AniList...")
	if err := browser.OpenURL(authURL); err != nil {
		fmt.Printf("Failed to open browser automatically. Please open the following URL manually:\n%s\n", authURL)
	}

	// Create a temporary file for the token
	tempFile, err := os.CreateTemp("", "anilist-token-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempFilePath := tempFile.Name()

	// Close the file so it can be reopened by the text editor
	tempFile.Close()

	fmt.Println("\nAfter authenticating, you'll be redirected to a page with the access token.")
	fmt.Printf("A text editor will open. Please paste the token from the URL into the file and save it.\n")

	// Try to use the user's preferred editor, falling back to nano
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano" // Default to nano if no EDITOR is set
	}

	cmd := exec.Command(editor, tempFilePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// If the default editor fails, try nano specifically
		if editor != "nano" {
			fmt.Println("Failed to open editor, trying nano...")
			cmd = exec.Command("nano", tempFilePath)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				// If nano fails too, try vim
				fmt.Println("Failed to open nano, trying vim...")
				cmd = exec.Command("vim", tempFilePath)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					return "", fmt.Errorf("failed to open any text editor: %w", err)
				}
			}
		} else {
			// If nano was the first choice and failed, try vim
			fmt.Println("Failed to open nano, trying vim...")
			cmd = exec.Command("vim", tempFilePath)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("failed to open any text editor: %w", err)
			}
		}
	}

	// Read the token from the file
	tokenBytes, err := os.ReadFile(tempFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read token file: %w", err)
	}

	// Clean up the temporary file
	os.Remove(tempFilePath)

	token := strings.TrimSpace(string(tokenBytes))
	if idx := strings.Index(token, "&"); idx > 0 {
		token = token[:idx]
	}

	return token, nil
}

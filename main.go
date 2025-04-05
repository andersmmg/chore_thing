package main

import (
	"andersmmg/chore_thing/grocy"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"

	_ "embed"
)

// Config holds our application configuration
type Config struct {
	GrocyURL     string `json:"grocy_url"`
	APIKey       string `json:"api_key"`
	Username     string `json:"username"`
	CheckTimeout int    `json:"check_timeout"` // in minutes
}

var userID int = -1
var hasOverdueChores bool = false

//go:embed assets/icon.ico
var normalIcon []byte

//go:embed assets/icon_warn.ico
var warningIcon []byte

// getDefaultConfig returns a config with default values
func getDefaultConfig() Config {
	return Config{
		GrocyURL:     "http://localhost:8080/api",
		APIKey:       "your-api-key-here",
		Username:     "andersmmg",
		CheckTimeout: 1,
	}
}

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "chore_thing", "config.json"), nil
}

// ensureConfigDir ensures the config directory exists
func ensureConfigDir() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "chore_thing")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	return nil
}

// loadConfig loads the configuration from a JSON file
func loadConfig() (Config, error) {
	var config Config

	// Get default config as a starting point
	defaultConfig := getDefaultConfig()

	// Get config path
	configPath, err := getConfigPath()
	if err != nil {
		return defaultConfig, fmt.Errorf("failed to determine config path: %w", err)
	}

	// Try to read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config doesn't exist, create it with defaults
			if err := saveConfig(defaultConfig); err != nil {
				return defaultConfig, fmt.Errorf("failed to create default config: %w", err)
			}
			log.Printf("Created default config at %s. Please edit it with your actual values.", configPath)
			beeep.Notify("No Config", "Created default config, please edit it with your actual values", "")
			// Open the config file directory location
			openFileBrowser(filepath.Dir(configPath))
			return defaultConfig, nil
		}
		return defaultConfig, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse existing JSON
	if err := json.Unmarshal(data, &config); err != nil {
		return defaultConfig, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Check for missing values and fill with defaults
	configUpdated := false

	if config.GrocyURL == "" {
		config.GrocyURL = defaultConfig.GrocyURL
		configUpdated = true
	}

	if config.APIKey == "" {
		config.APIKey = defaultConfig.APIKey
		configUpdated = true
	}

	if config.Username == "" {
		config.Username = defaultConfig.Username
		configUpdated = true
	}

	if config.CheckTimeout <= 0 {
		config.CheckTimeout = defaultConfig.CheckTimeout
		configUpdated = true
	}

	// If we had to add any default values, save the updated config
	if configUpdated {
		if err := saveConfig(config); err != nil {
			log.Printf("Warning: Failed to save updated config: %v", err)
		} else {
			log.Printf("Updated config with default values for missing fields")
		}
	}

	return config, nil
}

// saveConfig saves the configuration to a JSON file
func saveConfig(config Config) error {
	// Ensure config directory exists
	if err := ensureConfigDir(); err != nil {
		return err
	}

	// Get config path
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to determine config path: %w", err)
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func main() {
	// Start systray
	systray.Run(onReady, onExit)
}

func updateIcon() {
	// Set the appropriate icon based on whether there are overdue chores
	if hasOverdueChores {
		systray.SetIcon(warningIcon)
	} else {
		systray.SetIcon(normalIcon)
	}
}

func openBrowser(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	return err
}

func onReady() {
	systray.SetTitle("Chore Thing")
	systray.SetTooltip("Chore Thing - Track your chores")

	// Set the initial icon (normal icon since we haven't checked for chores yet)
	systray.SetIcon(normalIcon)

	// Menu items
	mCheck := systray.AddMenuItem("Check Chores", "Check for overdue chores")
	mToggleAutoCheck := systray.AddMenuItem("Auto Check: ON", "Toggle automatic checking")
	autoCheckEnabled := true
	mOpenWeb := systray.AddMenuItem("Open Web Overview", "Open chores overview in web browser")
	mSettings := systray.AddMenuItem("Settings", "View and edit settings")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	// Start ticker for periodic checking, initially with default timeout
	// The actual timeout will be loaded from config on each check
	initialConfig, _ := loadConfig()
	ticker := time.NewTicker(time.Duration(initialConfig.CheckTimeout) * time.Minute)

	// Do an initial check right away
	go checkChoresWithNotification()

	// Handle menu actions in a goroutine
	go func() {
		for {
			select {
			case <-mCheck.ClickedCh:
				fmt.Println("Manually checking chores...")
				go checkChoresWithNotification()

			case <-mToggleAutoCheck.ClickedCh:
				autoCheckEnabled = !autoCheckEnabled
				if autoCheckEnabled {
					mToggleAutoCheck.SetTitle("Auto Check: ON")
					// Always reload config to get the current timeout
					config, err := loadConfig()
					if err != nil {
						log.Printf("Error loading configuration: %v", err)
						config = getDefaultConfig()
					}
					ticker.Reset(time.Duration(config.CheckTimeout) * time.Minute)
					fmt.Println("Automatic checking enabled")
				} else {
					mToggleAutoCheck.SetTitle("Auto Check: OFF")
					ticker.Stop()
					fmt.Println("Automatic checking disabled")
				}

			case <-mOpenWeb.ClickedCh:
				// Get the base URL from the API URL (remove /api if it exists)
				config, err := loadConfig()
				if err != nil {
					log.Printf("Error loading configuration: %v", err)
					continue
				}

				baseURL := config.GrocyURL
				if len(baseURL) >= 4 && baseURL[len(baseURL)-4:] == "/api" {
					baseURL = baseURL[:len(baseURL)-4]
				}

				// Construct the URL
				webURL := fmt.Sprintf("%s/choresoverview", baseURL)
				if userID > 0 {
					webURL = fmt.Sprintf("%s?user=%d", webURL, userID)
				}

				// Open the URL in the default browser
				fmt.Printf("Opening Grocy web interface: %s\n", webURL)
				if err := openBrowser(webURL); err != nil {
					log.Printf("Error opening browser: %v", err)
				}

			case <-mSettings.ClickedCh:
				// Open the config file path in file browser
				configPath, _ := getConfigPath()
				if err := openFileBrowser(configPath); err != nil {
					log.Printf("Error opening file browser: %v", err)
				}

			case <-mQuit.ClickedCh:
				fmt.Println("Quit requested")
				ticker.Stop()
				systray.Quit()
				// Force exit after a brief delay as a backup
				go func() {
					time.Sleep(500 * time.Millisecond)
					os.Exit(0)
				}()
				return

			case <-ticker.C:
				if autoCheckEnabled {
					fmt.Println("Automatically checking chores...")
					// Always update ticker with latest timeout from config
					config, err := loadConfig()
					if err != nil {
						log.Printf("Error loading configuration: %v", err)
					} else {
						ticker.Reset(time.Duration(config.CheckTimeout) * time.Minute)
					}
					go checkChoresWithNotification()
				}
			}
		}
	}()
}

func onExit() {
	// Clean up here
	fmt.Println("Exiting...")
}

func openFileBrowser(path string) error {
	// Check os type
	if runtime.GOOS == "windows" {
		cmd := exec.Command("explorer", path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	} else if runtime.GOOS == "linux" {
		cmd := exec.Command("xdg-open", path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	} else if runtime.GOOS == "darwin" {
		cmd := exec.Command("open", path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return fmt.Errorf("unsupported OS")
}

func checkChoresWithNotification() {
	overdueChores := checkChores()

	// Update the icon based on overdue chores
	hasOverdueChores = len(overdueChores) > 0
	updateIcon()

	// Send notification if there are overdue chores
	if hasOverdueChores {
		title := "Chore Thing"
		message := fmt.Sprintf("You have %d overdue chores!", len(overdueChores))

		// Add the first few chores to the notification
		if len(overdueChores) > 0 {
			message += "\n"
			maxDisplay := 3 // Show at most 3 chores in the notification
			for i, chore := range overdueChores {
				if i >= maxDisplay {
					message += fmt.Sprintf("...and %d more", len(overdueChores)-maxDisplay)
					break
				}
				message += fmt.Sprintf("\n- %s", chore)
			}
		}

		err := beeep.Notify(title, message, "")
		if err != nil {
			log.Printf("Error sending notification: %v", err)
		}
	}
}

func checkChores() []string {
	// Load configuration - now loaded fresh every time this function is called
	config, err := loadConfig()
	if err != nil {
		log.Printf("Error loading configuration: %v", err)
		return nil
	}

	// Create client with config values
	client := grocy.NewGrocyClient(config.GrocyURL, config.APIKey)

	chores, err := client.GetChores()
	if err != nil {
		log.Printf("Error fetching chores: %v", err)
		return nil
	}

	currentTime := time.Now()
	username := config.Username // Use the configured username

	// Collect all overdue chores
	var overdueChores []string
	for _, chore := range chores {
		// Skip if the chore is not assigned to our user
		if chore.NextExecutionAssignedUser.Username != username {
			continue
		}

		userID = chore.NextExecutionAssignedToUserID

		// Parse the next execution time
		if chore.NextEstimatedExecutionTime == "" {
			continue // Skip if no execution time is set
		}

		// Use the correct date format: "2025-03-30 23:59:59"
		nextExecution, err := time.ParseInLocation("2006-01-02 15:04:05", chore.NextEstimatedExecutionTime, currentTime.Location())
		if err != nil {
			log.Printf("Error parsing time for chore %s: %v", chore.ChoreName, err)
			continue
		}

		// Check if the chore is overdue
		log.Printf("Current time: %s", currentTime)

		if nextExecution.Before(currentTime) {
			log.Printf("Overdue: %s", nextExecution)
			overdueChores = append(overdueChores, chore.ChoreName)
		}
	}

	// Print the summary in the requested format
	if len(overdueChores) > 0 {
		fmt.Printf("%d overdue tasks:\n", len(overdueChores))
		for _, choreName := range overdueChores {
			fmt.Printf("- %s\n", choreName)
		}
	} else {
		fmt.Println("No chores are currently overdue. Great job!")
	}

	return overdueChores
}

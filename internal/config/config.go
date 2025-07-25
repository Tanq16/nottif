package config

import (
	"encoding/json"
	"os"
	"sync"
)

// CronJob defines the structure for a single scheduled task.
type CronJob struct {
	ID       string `json:"id"`
	Message  string `json:"message"`
	Schedule string `json:"schedule"`
}

// Config defines the main application configuration.
type Config struct {
	Mu         sync.RWMutex `json:"-"` // Mutex for thread-safe access
	filePath   string       `json:"-"` // Path to the config file
	WebhookURL string       `json:"webhook_url"`
	CronJobs   []CronJob    `json:"cron_jobs"`
}

// LoadConfig reads the configuration file from disk.
func LoadConfig(path string) (*Config, error) {
	c := &Config{
		filePath: path,
	}

	// Check if the file exists, create it if it doesn't
	if _, err := os.Stat(path); os.IsNotExist(err) {
		c.CronJobs = []CronJob{} // Initialize with an empty slice
		return c, c.Save()
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Handle empty file case
	if len(file) == 0 {
		c.CronJobs = []CronJob{}
		return c, c.Save()
	}

	if err := json.Unmarshal(file, c); err != nil {
		return nil, err
	}

	return c, nil
}

// Save writes the current configuration to the disk.
func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.filePath, data, 0644)
}

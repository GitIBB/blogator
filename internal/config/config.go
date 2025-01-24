package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const configFileName = ".gatorconfig.json"

var configMutex sync.Mutex

// Config represents the structure of the config file
type Config struct {
	DbUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name,omitempty"`
}

func Read() (Config, error) {
	// initialize empty config struct
	var cfg Config

	// get full path to config file
	configPath, err := getConfigFilePath()
	if err != nil {
		return cfg, err
	}

	// open config file for reading
	file, err := os.Open(configPath)
	if err != nil {
		return cfg, err
	}

	// defer closing file
	defer file.Close()

	// create new JSON decoder to read file
	decoder := json.NewDecoder(file)
	// Decode JSON data into config struct
	err = decoder.Decode(&cfg)
	if err != nil {
		return cfg, err
	}

	// return populated config struct
	return cfg, nil
}

func (c *Config) SetUser(username string) error {
	c.CurrentUserName = username
	return c.Write()
}

func getConfigFilePath() (string, error) {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// join home dir with config filename
	configPath := filepath.Join(homeDir, configFileName)

	return configPath, nil
}

func (cfg *Config) Write() error {
	// mutex lock and deferral
	configMutex.Lock()
	defer configMutex.Unlock()

	// get full path to config file
	configPath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	// convert config struct to JSON bytes
	jsonData, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	// write JSON data to file with read/write permissions
	return os.WriteFile(configPath, jsonData, 0644)

}

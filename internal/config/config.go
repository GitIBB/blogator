package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configFileName = ".gatorconfig.json"

// Config represents the structure of the config file
type Config struct {
	DBUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name,omitempty"`
}

func read() (Config, error) {
	var cfg Config

	configPath, err := getConfigFilePath()
    if err != nil {
        return cfg, err
    }

	file, err := os.Open(configPath)
	if err != nil {
		return cfg, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

func getConfigFilePath() (string, error) {
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(homeDir, configFileName)

	return configPath, nil
}

write(cfg Config) error {

}
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/asabya/x-go/internal/db"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Usernames   []string `yaml:"usernames"`
	PostgresURL string   `yaml:"postgres_url"`
}

func main() {
	logger := log.New(os.Stdout, "[migrate] ", log.LstdFlags|log.Lshortfile)

	// Read config file
	configData, err := os.ReadFile("config.yaml")
	if err != nil {
		logger.Fatalf("Error reading config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		logger.Fatalf("Error parsing config file: %v", err)
	}

	if config.PostgresURL == "" {
		logger.Fatal("postgres_url is required in config.yaml")
	}

	if len(config.Usernames) == 0 {
		logger.Fatal("at least one username is required in config.yaml")
	}

	// Initialize database
	database, err := db.InitDB(config.PostgresURL, config.Usernames)
	if err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	fmt.Println("Database migration completed successfully!")
}

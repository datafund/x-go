package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/asabya/x-go/internal/handlers"
	"github.com/asabya/x-go/internal/tasks"
	"github.com/asabya/x-go/pkg/twitter"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq" // postgres driver
	"gopkg.in/yaml.v2"
)

type Config struct {
	Usernames   []string `yaml:"usernames"`
	PostgresURL string   `yaml:"postgres_url"`
}

func main() {
	// Set up logging
	logger := log.New(os.Stdout, "[twitter-http] ", log.LstdFlags|log.Lshortfile)

	// Get XGO path from environment variable or use default
	xgoPath := os.Getenv("XGO_PATH")
	if xgoPath == "" {
		logger.Fatalf("XGO_PATH is not set")
	}

	// Read config file from XGO_PATH
	configPath := filepath.Join(xgoPath, "config.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		logger.Fatalf("Error reading config file at %s: %v", configPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		logger.Fatalf("Error parsing config file: %v", err)
	}
	postgresURL := config.PostgresURL
	if postgresURL[len(postgresURL)-1] != '?' {
		postgresURL += "?"
	}
	if !strings.Contains(postgresURL, "sslmode=") {
		if postgresURL[len(postgresURL)-1] != '?' {
			postgresURL += "&"
		}
		postgresURL += "sslmode=disable"
	}

	// Connect to database
	database, err := sql.Open("postgres", postgresURL)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Test the connection
	if err := database.Ping(); err != nil {
		logger.Fatalf("Failed to ping database: %v", err)
	}

	// Create agent manager with account management
	agentManager, err := twitter.NewAgentManager(xgoPath)
	if err != nil {
		logger.Fatalf("Failed to create agent manager: %v", err)
	}

	// Check if at least one agent is logged in
	hasLoggedInAgent := false
	for i := 0; i < agentManager.GetAgentCount(); i++ {
		if agent, err := agentManager.GetAgent(i); err == nil && agent.IsLoggedIn() {
			hasLoggedInAgent = true
			break
		}
	}
	fmt.Println("hasLoggedInAgent", hasLoggedInAgent)

	// Start background tasks
	tasks.StartProfileUpdates(database, agentManager, logger)
	tasks.StartTweetUpdates(database, agentManager, logger)

	r := mux.NewRouter()

	// Basic endpoints that don't require login
	r.HandleFunc("/api/user/{username}/tweets", handlers.HandleGetUserTweetsWithManager(agentManager)).Methods("GET")
	r.HandleFunc("/api/user/{username}/profile", handlers.HandleGetProfileWithManager(agentManager)).Methods("GET")
	r.HandleFunc("/api/tweet/{id}", handlers.HandleGetTweetWithManager(agentManager)).Methods("GET")
	r.HandleFunc("/api/tweet/{id}/replies", handlers.HandleGetTweetRepliesWithManager(agentManager)).Methods("GET")
	r.HandleFunc("/api/search/tweets", handlers.HandleSearchTweetsInDB(database)).Methods("GET")
	r.HandleFunc("/api/users", handlers.HandleAddUser(database)).Methods("POST")

	// Endpoints that require login
	if hasLoggedInAgent {
		r.HandleFunc("/api/user/{username}/followers", handlers.HandleGetFollowersWithManager(agentManager)).Methods("GET")
		r.HandleFunc("/api/search", handlers.HandleSearchTweetsWithManager(agentManager)).Methods("GET")
		r.HandleFunc("/api/follow/{id}", handlers.HandleFollowUserWithManager(agentManager)).Methods("POST")
		r.HandleFunc("/api/unfollow/{id}", handlers.HandleUnfollowUserWithManager(agentManager)).Methods("POST")
		r.HandleFunc("/api/tweet", handlers.HandleCreateTweetWithManager(agentManager)).Methods("POST")
		r.HandleFunc("/api/tweet/{id}/like", handlers.HandleLikeTweetWithManager(agentManager)).Methods("POST")
		r.HandleFunc("/api/tweet/{id}/unlike", handlers.HandleUnlikeTweetWithManager(agentManager)).Methods("POST")
		r.HandleFunc("/api/tweet/{id}/retweet", handlers.HandleRetweetWithManager(agentManager)).Methods("POST")
	}

	// Add middleware for logging and recovery
	r.Use(handlers.LoggingMiddleware(logger))
	r.Use(mux.CORSMethodMiddleware(r))

	// Start the server
	addr := ":8080"
	logger.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
}

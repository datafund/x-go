package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/asabya/x-go/internal/handlers"
	"github.com/asabya/x-go/pkg/twitter"
	"github.com/gorilla/mux"
)

func main() {
	// Set up logging
	logger := log.New(os.Stdout, "[twitter-http] ", log.LstdFlags|log.Lshortfile)

	// Get XGO path from environment variable or use default
	xgoPath := os.Getenv("XGO_PATH")
	if xgoPath == "" {
		logger.Fatalf("XGO_PATH is not set")
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

	r := mux.NewRouter()

	// Basic endpoints that don't require login
	r.HandleFunc("/api/user/{username}/tweets", handlers.HandleGetUserTweetsWithManager(agentManager)).Methods("GET")
	r.HandleFunc("/api/user/{username}/profile", handlers.HandleGetProfileWithManager(agentManager)).Methods("GET")
	r.HandleFunc("/api/tweet/{id}", handlers.HandleGetTweetWithManager(agentManager)).Methods("GET")

	// Endpoints that require login
	if hasLoggedInAgent {
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

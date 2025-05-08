package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/asabya/x-go/pkg/twitter"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func loadCookies(cookieFile string) ([]*http.Cookie, error) {
	data, err := os.ReadFile(cookieFile)
	if err != nil {
		return nil, fmt.Errorf("error reading cookies: %v", err)
	}

	var cookies []*http.Cookie
	if err = json.Unmarshal(data, &cookies); err != nil {
		return nil, fmt.Errorf("error unmarshaling cookies: %v", err)
	}

	// Verify critical cookies are present
	var hasAuthToken, hasCSRFToken bool
	for _, cookie := range cookies {
		if cookie.Name == "auth_token" {
			hasAuthToken = true
		}
		if cookie.Name == "ct0" {
			hasCSRFToken = true
		}
	}

	if !hasAuthToken || !hasCSRFToken {
		return nil, fmt.Errorf("missing critical authentication cookies")
	}

	return cookies, nil
}

func main() {
	// Set up logging
	logger := log.New(os.Stdout, "[twitter-mcp] ", log.LstdFlags|log.Lshortfile)

	// Get XGO path from environment variable
	xgoPath := os.Getenv("XGO_PATH")
	if xgoPath == "" {
		logger.Fatalf("XGO_PATH is not set")
	}

	// Create agent manager
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
	logger.Printf("Has logged in agent: %v", hasLoggedInAgent)

	// Create a new MCP server with session configuration
	s := server.NewMCPServer(
		"Twitter Agent",
		"1.0.0",
		server.WithLogging(),
		server.WithRecovery(),
		server.WithToolCapabilities(true),
		server.WithToolHandlerMiddleware(func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
			return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return next(ctx, request)
			}
		}),
	)

	// Get the first agent to register tools
	firstAgent, err := agentManager.GetAgent(0)
	if err != nil {
		logger.Fatalf("Failed to get first agent: %v", err)
	}

	// Register tools from the first agent
	for _, tool := range firstAgent.GetTools() {
		s.AddTool(tool.Tool, tool.Handler)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("Shutting down server...")
		// No need to call Close() as it's handled by ServeStdio
	}()

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		logger.Printf("Server error: %v", err)
	}
}

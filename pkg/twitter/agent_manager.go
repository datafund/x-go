package twitter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/asabya/x-go/pkg/twitter/auth"
)

// Error definitions
var (
	ErrInvalidAgentIndex = errors.New("invalid agent index")
	ErrNoAccounts        = errors.New("no accounts found")
)

// AgentManager manages multiple Twitter agents and rotates between them for API calls
type AgentManager struct {
	agents      []*Agent
	mutex       sync.RWMutex
	index       uint32 // For round-robin agent selection
	authManager *auth.AccountManager
}

// NewAgentManager creates a new AgentManager with the provided agents
func NewAgentManager(xgoPath string) (*AgentManager, error) {
	authManager := auth.NewAccountManager(xgoPath)

	// Load accounts from accounts.json
	accounts, err := authManager.LoadAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to load accounts: %w", err)
	}

	if len(accounts) == 0 {
		return nil, ErrNoAccounts
	}

	agents := make([]*Agent, len(accounts))
	for i, account := range accounts {
		agent := NewAgent()

		// Try to load cookies first
		if authManager.CookiesExist(account.Username) {
			cookies, err := authManager.LoadCookies(account.Username)
			if err == nil {
				agent.SetCookies(cookies)
			}
		}

		// If not logged in (either no cookies or invalid cookies), try to login
		if !agent.IsLoggedIn() {
			if err := agent.Login(account.Username, account.Password); err != nil {
				return nil, fmt.Errorf("failed to login account %s: %w", account.Username, err)
			}

			// Save cookies after successful login
			cookies := agent.GetCookies()
			if err := authManager.SaveCookies(account.Username, cookies); err != nil {
				return nil, fmt.Errorf("failed to save cookies for account %s: %w", account.Username, err)
			}
		}

		agents[i] = agent
	}

	return &AgentManager{
		agents:      agents,
		index:       0,
		authManager: authManager,
	}, nil
}

// getNextAgent returns the next agent in a round-robin fashion
func (am *AgentManager) getNextAgent() *Agent {
	index := atomic.AddUint32(&am.index, 1)
	return am.agents[index%uint32(len(am.agents))]
}

// SetCookies sets the cookies for authentication for a specific agent
func (am *AgentManager) SetCookies(agentIndex int, cookies []*http.Cookie) error {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if agentIndex < 0 || agentIndex >= len(am.agents) {
		return ErrInvalidAgentIndex
	}

	am.agents[agentIndex].SetCookies(cookies)
	return nil
}

// GetUserTweets gets tweets from a specific user using the next available agent
func (am *AgentManager) GetUserTweets(ctx context.Context, username string, limit int, sortByOldest bool) (interface{}, error) {
	return am.getNextAgent().HandleGetUserTweets(ctx, username, limit, sortByOldest)
}

// GetProfile gets user profile information using the next available agent
func (am *AgentManager) GetProfile(ctx context.Context, username string) (interface{}, error) {
	return am.getNextAgent().HandleGetProfile(ctx, username)
}

// GetTweet gets a specific tweet using the next available agent
func (am *AgentManager) GetTweet(ctx context.Context, tweetID string) (interface{}, error) {
	return am.getNextAgent().HandleGetTweet(ctx, tweetID)
}

// SearchTweets searches for tweets using the next available agent
func (am *AgentManager) SearchTweets(ctx context.Context, query string, limit int) (interface{}, error) {
	return am.getNextAgent().HandleSearchTweets(ctx, query, limit)
}

// CreateTweet creates a new tweet using the next available agent
func (am *AgentManager) CreateTweet(ctx context.Context, text string, scheduleTime string) (interface{}, error) {
	return am.getNextAgent().HandleCreateTweet(ctx, text, scheduleTime)
}

// LikeTweet likes a tweet using the next available agent
func (am *AgentManager) LikeTweet(ctx context.Context, tweetID string) error {
	return am.getNextAgent().HandleLikeTweet(ctx, tweetID)
}

// UnlikeTweet unlikes a tweet using the next available agent
func (am *AgentManager) UnlikeTweet(ctx context.Context, tweetID string) error {
	return am.getNextAgent().HandleUnlikeTweet(ctx, tweetID)
}

// Retweet retweets a tweet using the next available agent
func (am *AgentManager) Retweet(ctx context.Context, tweetID string) error {
	return am.getNextAgent().HandleRetweet(ctx, tweetID)
}

// Follow follows a user using the next available agent
func (am *AgentManager) Follow(ctx context.Context, userID string) error {
	return am.getNextAgent().HandleFollow(ctx, userID)
}

// Unfollow unfollows a user using the next available agent
func (am *AgentManager) Unfollow(ctx context.Context, userID string) error {
	return am.getNextAgent().HandleUnfollow(ctx, userID)
}

// GetAgent returns the agent at the specified index
func (am *AgentManager) GetAgent(index int) (*Agent, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if index < 0 || index >= len(am.agents) {
		return nil, ErrInvalidAgentIndex
	}

	return am.agents[index], nil
}

// GetAgentCount returns the number of agents managed by the AgentManager
func (am *AgentManager) GetAgentCount() int {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	return len(am.agents)
}

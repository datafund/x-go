package twitter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/asabya/x-go/pkg/twitter/auth"
	"github.com/mark3labs/mcp-go/mcp"
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
	logger      *log.Logger
}

// NewAgentManager creates a new AgentManager with the provided agents
func NewAgentManager(xgoPath string) (*AgentManager, error) {
	authManager := auth.NewAccountManager(xgoPath)

	// Load accounts from accounts.json
	accounts, err := authManager.LoadAccounts()
	if err != nil {
		log.Printf("Failed to load accounts: %v", err)
		return nil, fmt.Errorf("failed to load accounts: %w", err)
	}

	if len(accounts) == 0 {
		log.Printf("No accounts found in accounts.json")
		return nil, ErrNoAccounts
	}

	agents := make([]*Agent, len(accounts))
	for i, account := range accounts {
		agent := NewAgent(account.Username)

		// Try to load cookies first
		if authManager.CookiesExist(account.Username) {
			cookies, err := authManager.LoadCookies(account.Username)
			if err == nil {
				agent.SetCookies(cookies)
				log.Printf("Loaded cookies for account: %s", account.Username)
			} else {
				log.Printf("Failed to load cookies for account %s: %v", account.Username, err)
			}
		}

		// If not logged in (either no cookies or invalid cookies), try to login
		if !agent.IsLoggedIn() {
			log.Printf("Attempting to login account: %s", account.Username)
			if err := agent.Login(account.Username, account.Password); err != nil {
				log.Printf("Failed to login account %s: %v", account.Username, err)
				return nil, fmt.Errorf("failed to login account %s: %w", account.Username, err)
			}
			log.Printf("Successfully logged in account: %s", account.Username)

			// Save cookies after successful login
			cookies := agent.GetCookies()
			if err := authManager.SaveCookies(account.Username, cookies); err != nil {
				log.Printf("Failed to save cookies for account %s: %v", account.Username, err)
				return nil, fmt.Errorf("failed to save cookies for account %s: %w", account.Username, err)
			}
			log.Printf("Saved cookies for account: %s", account.Username)
		}

		agents[i] = agent
	}

	return &AgentManager{
		agents:      agents,
		index:       0,
		authManager: authManager,
		logger:      log.Default(),
	}, nil
}

// getNextAgent returns the next agent in a round-robin fashion
func (am *AgentManager) getNextAgent() (*Agent, string) {
	index := atomic.AddUint32(&am.index, 1)
	agent := am.agents[index%uint32(len(am.agents))]
	am.logger.Printf("Selected agent: %s", agent.username)
	return agent, agent.username
}

// SetCookies sets the cookies for authentication for a specific agent
func (am *AgentManager) SetCookies(agentIndex int, cookies []*http.Cookie) error {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if agentIndex < 0 || agentIndex >= len(am.agents) {
		am.logger.Printf("Invalid agent index: %d", agentIndex)
		return ErrInvalidAgentIndex
	}

	am.agents[agentIndex].SetCookies(cookies)
	am.logger.Printf("Set cookies for agent index: %d", agentIndex)
	return nil
}

// GetUserTweets gets tweets from a specific user using the next available agent
func (am *AgentManager) GetUserTweets(ctx context.Context, username string, limit int, sortByOldest bool) (interface{}, string, error) {
	agent, agentUsername := am.getNextAgent()
	am.logger.Printf("Getting tweets for user %s using agent %s", username, agentUsername)

	result, err := agent.handleGetUserTweets(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name: "get_user_tweets",
			Arguments: map[string]interface{}{
				"username":       username,
				"limit":          float64(limit),
				"sort_by_oldest": sortByOldest,
			},
		},
	})
	if err != nil {
		am.logger.Printf("Error getting tweets for user %s: %v", username, err)
		return nil, agentUsername, err
	}
	if result.IsError {
		errMsg := result.Content[0].(*mcp.TextContent).Text
		am.logger.Printf("Error in response for user %s: %s", username, errMsg)
		return nil, agentUsername, fmt.Errorf(errMsg)
	}

	var data interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &data); err != nil {
		am.logger.Printf("Error unmarshaling response for user %s: %v", username, err)
		return nil, agentUsername, err
	}

	am.logger.Printf("Successfully retrieved tweets for user %s", username)
	return data, agentUsername, nil
}

// GetProfile gets user profile information using the next available agent
func (am *AgentManager) GetProfile(ctx context.Context, username string) (interface{}, string, error) {
	agent, agentUsername := am.getNextAgent()
	am.logger.Printf("Getting profile for user %s using agent %s", username, agentUsername)

	result, err := agent.handleGetProfile(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name: "get_profile",
			Arguments: map[string]interface{}{
				"username": username,
			},
		},
	})
	if err != nil {
		am.logger.Printf("Error getting profile for user %s: %v", username, err)
		return nil, agentUsername, err
	}
	if result.IsError {
		errMsg := result.Content[0].(*mcp.TextContent).Text
		am.logger.Printf("Error in response for profile %s: %s", username, errMsg)
		return nil, agentUsername, fmt.Errorf(errMsg)
	}

	var data interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &data); err != nil {
		am.logger.Printf("Error unmarshaling profile response for user %s: %v", username, err)
		return nil, agentUsername, err
	}

	am.logger.Printf("Successfully retrieved profile for user %s", username)
	return data, agentUsername, nil
}

// GetTweet gets a specific tweet using the next available agent
func (am *AgentManager) GetTweet(ctx context.Context, tweetID string) (interface{}, string, error) {
	agent, agentUsername := am.getNextAgent()
	am.logger.Printf("Getting tweet %s using agent %s", tweetID, agentUsername)

	result, err := agent.handleGetTweet(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name: "get_tweet",
			Arguments: map[string]interface{}{
				"tweet_id": tweetID,
			},
		},
	})
	if err != nil {
		am.logger.Printf("Error getting tweet %s: %v", tweetID, err)
		return nil, agentUsername, err
	}
	if result.IsError {
		errMsg := result.Content[0].(*mcp.TextContent).Text
		am.logger.Printf("Error in response for tweet %s: %s", tweetID, errMsg)
		return nil, agentUsername, fmt.Errorf(errMsg)
	}

	var data interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &data); err != nil {
		am.logger.Printf("Error unmarshaling tweet response for %s: %v", tweetID, err)
		return nil, agentUsername, err
	}

	am.logger.Printf("Successfully retrieved tweet %s", tweetID)
	return data, agentUsername, nil
}

// SearchTweets searches for tweets using the next available agent
func (am *AgentManager) SearchTweets(ctx context.Context, query string, limit int) (interface{}, string, error) {
	agent, agentUsername := am.getNextAgent()
	am.logger.Printf("Searching tweets with query '%s' using agent %s", query, agentUsername)

	result, err := agent.handleSearchTweets(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name: "search_tweets",
			Arguments: map[string]interface{}{
				"query": query,
				"limit": float64(limit),
			},
		},
	})
	if err != nil {
		am.logger.Printf("Error searching tweets with query '%s': %v", query, err)
		return nil, agentUsername, err
	}
	if result.IsError {
		errMsg := result.Content[0].(*mcp.TextContent).Text
		am.logger.Printf("Error in response for search query '%s': %s", query, errMsg)
		return nil, agentUsername, fmt.Errorf(errMsg)
	}

	var data interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &data); err != nil {
		am.logger.Printf("Error unmarshaling search response for query '%s': %v", query, err)
		return nil, agentUsername, err
	}

	am.logger.Printf("Successfully searched tweets with query '%s'", query)
	return data, agentUsername, nil
}

// CreateTweet creates a new tweet using the next available agent
func (am *AgentManager) CreateTweet(ctx context.Context, text string, scheduleTime string) (interface{}, string, error) {
	agent, agentUsername := am.getNextAgent()
	am.logger.Printf("Creating tweet using agent %s", agentUsername)

	result, err := agent.handleCreateTweet(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name: "create_tweet",
			Arguments: map[string]interface{}{
				"text":          text,
				"schedule_time": scheduleTime,
			},
		},
	})
	if err != nil {
		am.logger.Printf("Error creating tweet: %v", err)
		return nil, agentUsername, err
	}
	if result.IsError {
		errMsg := result.Content[0].(*mcp.TextContent).Text
		am.logger.Printf("Error in response for creating tweet: %s", errMsg)
		return nil, agentUsername, fmt.Errorf(errMsg)
	}

	var data interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &data); err != nil {
		am.logger.Printf("Error unmarshaling create tweet response: %v", err)
		return nil, agentUsername, err
	}

	am.logger.Printf("Successfully created tweet")
	return data, agentUsername, nil
}

// LikeTweet likes a tweet using the next available agent
func (am *AgentManager) LikeTweet(ctx context.Context, tweetID string) (string, error) {
	agent, agentUsername := am.getNextAgent()
	am.logger.Printf("Liking tweet %s using agent %s", tweetID, agentUsername)

	result, err := agent.handleLikeTweet(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name: "like_tweet",
			Arguments: map[string]interface{}{
				"tweet_id": tweetID,
			},
		},
	})
	if err != nil {
		am.logger.Printf("Error liking tweet %s: %v", tweetID, err)
		return agentUsername, err
	}
	if result.IsError {
		errMsg := result.Content[0].(*mcp.TextContent).Text
		am.logger.Printf("Error in response for liking tweet %s: %s", tweetID, errMsg)
		return agentUsername, fmt.Errorf(errMsg)
	}

	am.logger.Printf("Successfully liked tweet %s", tweetID)
	return agentUsername, nil
}

// UnlikeTweet unlikes a tweet using the next available agent
func (am *AgentManager) UnlikeTweet(ctx context.Context, tweetID string) (string, error) {
	agent, agentUsername := am.getNextAgent()
	am.logger.Printf("Unliking tweet %s using agent %s", tweetID, agentUsername)

	result, err := agent.handleUnlikeTweet(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name: "unlike_tweet",
			Arguments: map[string]interface{}{
				"tweet_id": tweetID,
			},
		},
	})
	if err != nil {
		am.logger.Printf("Error unliking tweet %s: %v", tweetID, err)
		return agentUsername, err
	}
	if result.IsError {
		errMsg := result.Content[0].(*mcp.TextContent).Text
		am.logger.Printf("Error in response for unliking tweet %s: %s", tweetID, errMsg)
		return agentUsername, fmt.Errorf(errMsg)
	}

	am.logger.Printf("Successfully unliked tweet %s", tweetID)
	return agentUsername, nil
}

// Retweet retweets a tweet using the next available agent
func (am *AgentManager) Retweet(ctx context.Context, tweetID string) (string, error) {
	agent, agentUsername := am.getNextAgent()
	am.logger.Printf("Retweeting tweet %s using agent %s", tweetID, agentUsername)

	result, err := agent.handleRetweet(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name: "retweet",
			Arguments: map[string]interface{}{
				"tweet_id": tweetID,
			},
		},
	})
	if err != nil {
		am.logger.Printf("Error retweeting tweet %s: %v", tweetID, err)
		return agentUsername, err
	}
	if result.IsError {
		errMsg := result.Content[0].(*mcp.TextContent).Text
		am.logger.Printf("Error in response for retweeting tweet %s: %s", tweetID, errMsg)
		return agentUsername, fmt.Errorf(errMsg)
	}

	am.logger.Printf("Successfully retweeted tweet %s", tweetID)
	return agentUsername, nil
}

// Follow follows a user using the next available agent
func (am *AgentManager) Follow(ctx context.Context, userID string) (string, error) {
	agent, agentUsername := am.getNextAgent()
	am.logger.Printf("Following user %s using agent %s", userID, agentUsername)

	result, err := agent.handleFollowUser(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name: "follow",
			Arguments: map[string]interface{}{
				"user_id": userID,
			},
		},
	})
	if err != nil {
		am.logger.Printf("Error following user %s: %v", userID, err)
		return agentUsername, err
	}
	if result.IsError {
		errMsg := result.Content[0].(*mcp.TextContent).Text
		am.logger.Printf("Error in response for following user %s: %s", userID, errMsg)
		return agentUsername, fmt.Errorf(errMsg)
	}

	am.logger.Printf("Successfully followed user %s", userID)
	return agentUsername, nil
}

// Unfollow unfollows a user using the next available agent
func (am *AgentManager) Unfollow(ctx context.Context, userID string) (string, error) {
	agent, agentUsername := am.getNextAgent()
	am.logger.Printf("Unfollowing user %s using agent %s", userID, agentUsername)

	result, err := agent.handleUnfollowUser(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name: "unfollow",
			Arguments: map[string]interface{}{
				"user_id": userID,
			},
		},
	})
	if err != nil {
		am.logger.Printf("Error unfollowing user %s: %v", userID, err)
		return agentUsername, err
	}
	if result.IsError {
		errMsg := result.Content[0].(*mcp.TextContent).Text
		am.logger.Printf("Error in response for unfollowing user %s: %s", userID, errMsg)
		return agentUsername, fmt.Errorf(errMsg)
	}

	am.logger.Printf("Successfully unfollowed user %s", userID)
	return agentUsername, nil
}

// GetAgent returns the agent at the specified index
func (am *AgentManager) GetAgent(index int) (*Agent, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if index < 0 || index >= len(am.agents) {
		am.logger.Printf("Invalid agent index requested: %d", index)
		return nil, ErrInvalidAgentIndex
	}

	am.logger.Printf("Retrieved agent at index %d", index)
	return am.agents[index], nil
}

// GetAgentCount returns the number of agents managed by the AgentManager
func (am *AgentManager) GetAgentCount() int {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	count := len(am.agents)
	am.logger.Printf("Current agent count: %d", count)
	return count
}

// GetFollowers gets followers of a specific user using the next available agent
func (am *AgentManager) GetFollowers(ctx context.Context, username string, limit int, cursor string) (interface{}, string, error) {
	agent, agentUsername := am.getNextAgent()
	am.logger.Printf("Getting followers for user %s using agent %s", username, agentUsername)

	result, err := agent.handleGetFollowers(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name: "get_followers",
			Arguments: map[string]interface{}{
				"username": username,
				"limit":    float64(limit),
				"cursor":   cursor,
			},
		},
	})
	if err != nil {
		am.logger.Printf("Error getting followers for user %s: %v", username, err)
		return nil, agentUsername, err
	}
	if result.IsError {
		errMsg := result.Content[0].(*mcp.TextContent).Text
		am.logger.Printf("Error in response for followers %s: %s", username, errMsg)
		return nil, agentUsername, fmt.Errorf(errMsg)
	}

	var data interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &data); err != nil {
		am.logger.Printf("Error unmarshaling followers response for user %s: %v", username, err)
		return nil, agentUsername, err
	}

	am.logger.Printf("Successfully retrieved followers for user %s", username)
	return data, agentUsername, nil
}

// GetTweetReplies gets replies to a specific tweet using the next available agent
func (am *AgentManager) GetTweetReplies(ctx context.Context, tweetID string, cursor string) (interface{}, string, error) {
	agent, agentUsername := am.getNextAgent()
	am.logger.Printf("Getting replies for tweet %s using agent %s", tweetID, agentUsername)

	result, err := agent.handleGetTweetReplies(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name: "get_tweet_replies",
			Arguments: map[string]interface{}{
				"tweet_id": tweetID,
				"cursor":   cursor,
			},
		},
	})
	if err != nil {
		am.logger.Printf("Error getting replies for tweet %s: %v", tweetID, err)
		return nil, agentUsername, err
	}
	if result.IsError {
		errMsg := result.Content[0].(*mcp.TextContent).Text
		am.logger.Printf("Error in response for tweet replies %s: %s", tweetID, errMsg)
		return nil, agentUsername, fmt.Errorf(errMsg)
	}

	var data interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &data); err != nil {
		am.logger.Printf("Error unmarshaling replies response for tweet %s: %v", tweetID, err)
		return nil, agentUsername, err
	}

	am.logger.Printf("Successfully retrieved replies for tweet %s", tweetID)
	return data, agentUsername, nil
}

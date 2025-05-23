package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	twitterscraper "github.com/imperatrona/twitter-scraper"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Scraper interface for Twitter operations
type Scraper interface {
	IsLoggedIn() bool
	SetCookies([]*http.Cookie)
	GetProfile(ctx context.Context, username string) (*twitterscraper.Profile, error)
	GetTweets(ctx context.Context, username string, maxTweetsNb int) <-chan *twitterscraper.TweetResult
	GetTweet(ctx context.Context, id string) (*twitterscraper.Tweet, error)
	GetTweetReplies(id string, cursor string) ([]*twitterscraper.Tweet, []*twitterscraper.ThreadCursor, error)
	SearchTweets(ctx context.Context, query string, maxTweetsNb int) <-chan *twitterscraper.TweetResult
	Tweet(ctx context.Context, text string) (*twitterscraper.Tweet, error)
	LikeTweet(ctx context.Context, id string) error
	UnlikeTweet(ctx context.Context, id string) error
	CreateRetweet(ctx context.Context, id string) error
	CreateScheduledTweet(ctx context.Context, text string, scheduleTime string) error
	Follow(ctx context.Context, id string) error
	Unfollow(ctx context.Context, id string) error
	Login(credentials ...string) error
	GetCookies() []*http.Cookie
	FetchFollowers(username string, maxUsersNbr int, cursor string) ([]*twitterscraper.Profile, string, error)
}

// Agent represents a Twitter MCP agent
type Agent struct {
	scraper  Scraper
	limiter  *rateLimiter
	username string
}

// NewAgent creates a new Twitter MCP agent
func NewAgent(username string) *Agent {
	return &Agent{
		scraper:  newScraperWrapper(),
		limiter:  newRateLimiter(),
		username: username,
	}
}

// SetCookies sets the cookies for authentication
func (a *Agent) SetCookies(cookies []*http.Cookie) {
	a.scraper.SetCookies(cookies)
}

// GetCookies returns the current cookies for the agent
func (a *Agent) GetCookies() []*http.Cookie {
	return a.scraper.GetCookies()
}

// GetTools returns the list of available tools
func (a *Agent) GetTools() []server.ServerTool {
	// Basic tools that don't require login
	tools := []server.ServerTool{
		{
			Tool: mcp.Tool{
				Name:        "get_user_tweets",
				Description: "Get tweets from a specific user",
				InputSchema: mcp.ToolInputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"username": map[string]interface{}{
							"type":        "string",
							"description": "Twitter username",
						},
						"limit": map[string]interface{}{
							"type":        "number",
							"description": "Maximum number of tweets to fetch",
							"default":     50,
						},
						"sort_by_oldest": map[string]interface{}{
							"type":        "boolean",
							"description": "Sort tweets by oldest",
						},
					},
					Required: []string{"username"},
				},
				Annotations: mcp.ToolAnnotation{
					Title:         "Get User Tweets",
					ReadOnlyHint:  BoolPtr(true),
					OpenWorldHint: BoolPtr(true),
				},
			},
			Handler: a.handleGetUserTweets,
		},
		{
			Tool: mcp.Tool{
				Name:        "get_profile",
				Description: "Get user profile information",
				InputSchema: mcp.ToolInputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"username": map[string]interface{}{
							"type":        "string",
							"description": "Twitter username",
						},
					},
					Required: []string{"username"},
				},
				Annotations: mcp.ToolAnnotation{
					Title:         "Get User Profile",
					ReadOnlyHint:  BoolPtr(true),
					OpenWorldHint: BoolPtr(true),
				},
			},
			Handler: a.handleGetProfile,
		},
		{
			Tool: mcp.Tool{
				Name:        "get_tweet",
				Description: "Get a specific tweet by ID",
				InputSchema: mcp.ToolInputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"tweet_id": map[string]interface{}{
							"type":        "string",
							"description": "Tweet ID",
						},
					},
					Required: []string{"tweet_id"},
				},
				Annotations: mcp.ToolAnnotation{
					Title:         "Get Tweet",
					ReadOnlyHint:  BoolPtr(true),
					OpenWorldHint: BoolPtr(true),
				},
			},
			Handler: a.handleGetTweet,
		},
		{
			Tool: mcp.Tool{
				Name:        "get_followers",
				Description: "Get followers of a specific user",
				InputSchema: mcp.ToolInputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"username": map[string]interface{}{
							"type":        "string",
							"description": "Twitter username",
						},
						"limit": map[string]interface{}{
							"type":        "number",
							"description": "Maximum number of followers to fetch",
							"default":     50,
						},
						"cursor": map[string]interface{}{
							"type":        "string",
							"description": "Cursor for pagination",
						},
					},
					Required: []string{"username"},
				},
				Annotations: mcp.ToolAnnotation{
					Title:         "Get User Followers",
					ReadOnlyHint:  BoolPtr(true),
					OpenWorldHint: BoolPtr(true),
				},
			},
			Handler: a.handleGetFollowers,
		},
		{
			Tool: mcp.Tool{
				Name:        "get_tweet_replies",
				Description: "Get replies to a specific tweet",
				InputSchema: mcp.ToolInputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"tweet_id": map[string]interface{}{
							"type":        "string",
							"description": "ID of the tweet to get replies for",
						},
						"cursor": map[string]interface{}{
							"type":        "string",
							"description": "Cursor for pagination",
						},
					},
					Required: []string{"tweet_id"},
				},
				Annotations: mcp.ToolAnnotation{
					Title:         "Get Tweet Replies",
					ReadOnlyHint:  BoolPtr(true),
					OpenWorldHint: BoolPtr(true),
				},
			},
			Handler: a.handleGetTweetReplies,
		},
	}

	// Add tools that require login only if logged in
	if a.scraper.IsLoggedIn() {
		tools = append(tools,
			server.ServerTool{
				Tool: mcp.Tool{
					Name:        "search_tweets",
					Description: "Search for tweets",
					InputSchema: mcp.ToolInputSchema{
						Type: "object",
						Properties: map[string]interface{}{
							"query": map[string]interface{}{
								"type":        "string",
								"description": "Search query",
							},
							"limit": map[string]interface{}{
								"type":        "number",
								"description": "Maximum number of tweets to fetch",
								"default":     50,
							},
						},
						Required: []string{"query"},
					},
					Annotations: mcp.ToolAnnotation{
						Title:         "Search Tweets",
						ReadOnlyHint:  BoolPtr(true),
						OpenWorldHint: BoolPtr(true),
					},
				},
				Handler: a.handleSearchTweets,
			},
			server.ServerTool{
				Tool: mcp.Tool{
					Name:        "create_tweet",
					Description: "Create a new tweet",
					InputSchema: mcp.ToolInputSchema{
						Type: "object",
						Properties: map[string]interface{}{
							"text": map[string]interface{}{
								"type":        "string",
								"description": "Tweet text content",
							},
							"schedule_time": map[string]interface{}{
								"type":        "string",
								"description": "Optional ISO8601 timestamp for scheduled tweets",
							},
						},
						Required: []string{"text"},
					},
					Annotations: mcp.ToolAnnotation{
						Title: "Create Tweet",
					},
				},
				Handler: a.handleCreateTweet,
			},
			server.ServerTool{
				Tool: mcp.Tool{
					Name:        "like_tweet",
					Description: "Like a tweet",
					InputSchema: mcp.ToolInputSchema{
						Type: "object",
						Properties: map[string]interface{}{
							"tweet_id": map[string]interface{}{
								"type":        "string",
								"description": "ID of the tweet to like",
							},
						},
						Required: []string{"tweet_id"},
					},
					Annotations: mcp.ToolAnnotation{
						Title: "Like Tweet",
					},
				},
				Handler: a.handleLikeTweet,
			},
			server.ServerTool{
				Tool: mcp.Tool{
					Name:        "unlike_tweet",
					Description: "Unlike a tweet",
					InputSchema: mcp.ToolInputSchema{
						Type: "object",
						Properties: map[string]interface{}{
							"tweet_id": map[string]interface{}{
								"type":        "string",
								"description": "ID of the tweet to unlike",
							},
						},
						Required: []string{"tweet_id"},
					},
					Annotations: mcp.ToolAnnotation{
						Title: "Unlike Tweet",
					},
				},
				Handler: a.handleUnlikeTweet,
			},
			server.ServerTool{
				Tool: mcp.Tool{
					Name:        "retweet",
					Description: "Retweet a tweet",
					InputSchema: mcp.ToolInputSchema{
						Type: "object",
						Properties: map[string]interface{}{
							"tweet_id": map[string]interface{}{
								"type":        "string",
								"description": "ID of the tweet to retweet",
							},
						},
						Required: []string{"tweet_id"},
					},
					Annotations: mcp.ToolAnnotation{
						Title: "Retweet",
					},
				},
				Handler: a.handleRetweet,
			},
		)
	}

	return tools
}

// Tool handlers
func (a *Agent) handleGetUserTweets(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	username, ok := request.Params.Arguments["username"].(string)
	if !ok || username == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "username parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	limit := 50
	if limitVal, ok := request.Params.Arguments["limit"].(float64); ok {
		limit = int(limitVal)
	}

	// Wait for rate limit
	if err := a.limiter.waitForEndpoint(ctx, "get_user_tweets"); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("rate limit error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	tweets := a.scraper.GetTweets(ctx, username, limit)
	var results []twitterscraper.TweetResult

	for tweet := range tweets {
		if tweet.Error != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("error getting tweets: %v", tweet.Error),
					},
				},
				IsError: true,
			}, nil
		}
		results = append(results, *tweet)
	}

	jsonData, err := json.Marshal(results)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error marshaling results: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Type: "text",
				Text: string(jsonData),
			},
		},
	}, nil
}

func (a *Agent) handleGetProfile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	username, ok := request.Params.Arguments["username"].(string)
	if !ok || username == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "username parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	// Wait for rate limit
	if err := a.limiter.waitForEndpoint(ctx, "get_profile"); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("rate limit error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	profile, err := a.scraper.GetProfile(ctx, username)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error getting profile: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	jsonData, err := json.Marshal(profile)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error marshaling results: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Type: "text",
				Text: string(jsonData),
			},
		},
	}, nil
}

func (a *Agent) handleGetTweet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tweetID, ok := request.Params.Arguments["tweet_id"].(string)
	if !ok || tweetID == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "tweet_id parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	// Wait for rate limit
	if err := a.limiter.waitForEndpoint(ctx, "get_tweet"); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("rate limit error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	tweet, err := a.scraper.GetTweet(ctx, tweetID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error getting tweet: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	jsonData, err := json.Marshal(tweet)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error marshaling results: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Type: "text",
				Text: string(jsonData),
			},
		},
	}, nil
}

func (a *Agent) handleSearchTweets(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !a.scraper.IsLoggedIn() {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "This tool requires login. Please provide Twitter cookies to use this tool.",
				},
			},
			IsError: true,
		}, nil
	}

	query, ok := request.Params.Arguments["query"].(string)
	if !ok || query == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "query parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	limit := 50
	if limitVal, ok := request.Params.Arguments["limit"].(float64); ok {
		limit = int(limitVal)
	}

	// Wait for rate limit
	if err := a.limiter.waitForEndpoint(ctx, "search_tweets"); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("rate limit error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	tweets := a.scraper.SearchTweets(ctx, query, limit)
	var results []map[string]interface{}

	for tweet := range tweets {
		if tweet.Error != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("error searching tweets: %v", tweet.Error),
					},
				},
				IsError: true,
			}, nil
		}
		results = append(results, map[string]interface{}{
			"id":        tweet.ID,
			"text":      tweet.Text,
			"likes":     tweet.Likes,
			"retweets":  tweet.Retweets,
			"replies":   tweet.Replies,
			"timestamp": tweet.TimeParsed,
			"author": map[string]interface{}{
				"username": tweet.Username,
				"name":     tweet.Name,
			},
		})
	}

	jsonData, err := json.Marshal(results)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error marshaling results: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Type: "text",
				Text: string(jsonData),
			},
		},
	}, nil
}

func (a *Agent) handleCreateTweet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !a.scraper.IsLoggedIn() {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "This tool requires login. Please provide Twitter cookies to use this tool.",
				},
			},
			IsError: true,
		}, nil
	}

	text, ok := request.Params.Arguments["text"].(string)
	if !ok || text == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "text parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	// Wait for rate limit
	if err := a.limiter.waitForEndpoint(ctx, "create_tweet"); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("rate limit error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	tweet, err := a.scraper.Tweet(ctx, text)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error creating tweet: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	jsonData, err := json.Marshal(tweet)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error marshaling results: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Type: "text",
				Text: string(jsonData),
			},
		},
	}, nil
}

func (a *Agent) handleLikeTweet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !a.scraper.IsLoggedIn() {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "This tool requires login. Please provide Twitter cookies to use this tool.",
				},
			},
			IsError: true,
		}, nil
	}

	tweetID, ok := request.Params.Arguments["tweet_id"].(string)
	if !ok || tweetID == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "tweet_id is required",
				},
			},
			IsError: true,
		}, nil
	}

	// Wait for rate limit
	if err := a.limiter.waitForEndpoint(ctx, "like_tweet"); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("rate limit error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	err := a.scraper.LikeTweet(ctx, tweetID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error liking tweet: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Type: "text",
				Text: "Tweet liked successfully",
			},
		},
	}, nil
}

func (a *Agent) handleFollowUser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !a.scraper.IsLoggedIn() {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "This tool requires login. Please provide Twitter cookies to use this tool.",
				},
			},
			IsError: true,
		}, nil
	}

	userID, ok := request.Params.Arguments["user_id"].(string)
	if !ok || userID == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "user_id is required",
				},
			},
			IsError: true,
		}, nil
	}

	// Wait for rate limit
	if err := a.limiter.waitForEndpoint(ctx, "follow_user"); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("rate limit error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	err := a.scraper.Follow(ctx, userID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error following user: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Type: "text",
				Text: "User followed successfully",
			},
		},
	}, nil
}

func (a *Agent) handleUnfollowUser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !a.scraper.IsLoggedIn() {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "This tool requires login. Please provide Twitter cookies to use this tool.",
				},
			},
			IsError: true,
		}, nil
	}

	userID, ok := request.Params.Arguments["user_id"].(string)
	if !ok || userID == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "user_id is required",
				},
			},
			IsError: true,
		}, nil
	}

	// Wait for rate limit
	if err := a.limiter.waitForEndpoint(ctx, "unfollow_user"); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("rate limit error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	err := a.scraper.Unfollow(ctx, userID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error unfollowing user: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Type: "text",
				Text: "User unfollowed successfully",
			},
		},
	}, nil
}

func (a *Agent) handleUnlikeTweet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !a.scraper.IsLoggedIn() {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "This tool requires login. Please provide Twitter cookies to use this tool.",
				},
			},
			IsError: true,
		}, nil
	}

	tweetID, ok := request.Params.Arguments["tweet_id"].(string)
	if !ok || tweetID == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "tweet_id is required",
				},
			},
			IsError: true,
		}, nil
	}

	// Wait for rate limit
	if err := a.limiter.waitForEndpoint(ctx, "unlike_tweet"); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("rate limit error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	err := a.scraper.UnlikeTweet(ctx, tweetID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error unliking tweet: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Type: "text",
				Text: "Tweet unliked successfully",
			},
		},
	}, nil
}

func (a *Agent) handleRetweet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !a.scraper.IsLoggedIn() {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "This tool requires login. Please provide Twitter cookies to use this tool.",
				},
			},
			IsError: true,
		}, nil
	}

	tweetID, ok := request.Params.Arguments["tweet_id"].(string)
	if !ok || tweetID == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "tweet_id is required",
				},
			},
			IsError: true,
		}, nil
	}

	// Wait for rate limit
	if err := a.limiter.waitForEndpoint(ctx, "retweet"); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("rate limit error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	err := a.scraper.CreateRetweet(ctx, tweetID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error retweeting: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Type: "text",
				Text: "Tweet retweeted successfully",
			},
		},
	}, nil
}

func (a *Agent) handleGetFollowers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	username, ok := request.Params.Arguments["username"].(string)
	if !ok || username == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "username parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	limit := 50
	if limitVal, ok := request.Params.Arguments["limit"].(float64); ok {
		limit = int(limitVal)
	}

	cursor := ""
	if cursorVal, ok := request.Params.Arguments["cursor"].(string); ok {
		cursor = cursorVal
	}

	// Wait for rate limit
	if err := a.limiter.waitForEndpoint(ctx, "get_followers"); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("rate limit error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	followers, nextCursor, err := a.scraper.FetchFollowers(username, limit, cursor)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error getting followers: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	result := map[string]interface{}{
		"followers":   followers,
		"next_cursor": nextCursor,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error marshaling results: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Type: "text",
				Text: string(jsonData),
			},
		},
	}, nil
}

func (a *Agent) handleGetTweetReplies(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tweetID, ok := request.Params.Arguments["tweet_id"].(string)
	if !ok || tweetID == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: "tweet_id parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	cursor := ""
	if cursorVal, ok := request.Params.Arguments["cursor"].(string); ok {
		cursor = cursorVal
	}

	// Wait for rate limit
	if err := a.limiter.waitForEndpoint(ctx, "get_tweet_replies"); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("rate limit error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	replies, nextCursor, err := a.scraper.GetTweetReplies(tweetID, cursor)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error getting tweet replies: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Create simplified tweet structures to avoid circular references
	type SimplifiedTweet struct {
		ID         string    `json:"id"`
		Text       string    `json:"text"`
		Username   string    `json:"username"`
		Name       string    `json:"name"`
		Likes      int       `json:"likes"`
		Retweets   int       `json:"retweets"`
		Replies    int       `json:"replies"`
		TimeParsed time.Time `json:"timestamp"`
	}

	simplifiedReplies := make([]SimplifiedTweet, 0, len(replies))
	for _, reply := range replies {
		simplifiedReplies = append(simplifiedReplies, SimplifiedTweet{
			ID:         reply.ID,
			Text:       reply.Text,
			Username:   reply.Username,
			Name:       reply.Name,
			Likes:      reply.Likes,
			Retweets:   reply.Retweets,
			Replies:    reply.Replies,
			TimeParsed: reply.TimeParsed,
		})
	}

	// Create simplified cursor structure
	type SimplifiedCursor struct {
		FocalTweetID string `json:"focal_tweet_id"`
		ThreadID     string `json:"thread_id"`
		Cursor       string `json:"cursor"`
		CursorType   string `json:"cursor_type"`
	}

	simplifiedCursors := make([]SimplifiedCursor, 0, len(nextCursor))
	for _, cursor := range nextCursor {
		simplifiedCursors = append(simplifiedCursors, SimplifiedCursor{
			FocalTweetID: cursor.FocalTweetID,
			ThreadID:     cursor.ThreadID,
			Cursor:       cursor.Cursor,
			CursorType:   cursor.CursorType,
		})
	}

	result := struct {
		Replies    []SimplifiedTweet  `json:"replies"`
		NextCursor []SimplifiedCursor `json:"next_cursor"`
	}{
		Replies:    simplifiedReplies,
		NextCursor: simplifiedCursors,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("error marshaling results: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Type: "text",
				Text: string(jsonData),
			},
		},
	}, nil
}

// Login logs in to Twitter using the provided credentials
func (a *Agent) Login(credentials ...string) error {
	return a.scraper.Login(credentials...)
}

// IsLoggedIn returns whether the agent is logged in
func (a *Agent) IsLoggedIn() bool {
	return a.scraper.IsLoggedIn()
}

// HandleGetUserTweets handles getting user tweets
func (a *Agent) HandleGetUserTweets(ctx context.Context, username string, limit int, sortByOldest bool) (interface{}, error) {
	result, err := a.handleGetUserTweets(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"username":       username,
				"limit":          float64(limit),
				"sort_by_oldest": sortByOldest,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if result.IsError {
		return nil, fmt.Errorf(result.Content[0].(*mcp.TextContent).Text)
	}
	var data interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &data); err != nil {
		return nil, err
	}
	return data, nil
}

// HandleGetProfile handles getting user profile
func (a *Agent) HandleGetProfile(ctx context.Context, username string) (interface{}, error) {
	result, err := a.handleGetProfile(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"username": username,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if result.IsError {
		return nil, fmt.Errorf(result.Content[0].(*mcp.TextContent).Text)
	}
	var data interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &data); err != nil {
		return nil, err
	}
	return data, nil
}

// HandleGetTweet handles getting a tweet
func (a *Agent) HandleGetTweet(ctx context.Context, tweetID string) (interface{}, error) {
	result, err := a.handleGetTweet(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"tweet_id": tweetID,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if result.IsError {
		return nil, fmt.Errorf(result.Content[0].(*mcp.TextContent).Text)
	}
	var data interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &data); err != nil {
		return nil, err
	}
	return data, nil
}

// HandleGetFollowers handles getting user followers
func (a *Agent) HandleGetFollowers(ctx context.Context, username string, limit int, cursor string) (interface{}, error) {
	result, err := a.handleGetFollowers(ctx, mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"username": username,
				"limit":    float64(limit),
				"cursor":   cursor,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if result.IsError {
		return nil, fmt.Errorf(result.Content[0].(*mcp.TextContent).Text)
	}
	var data interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &data); err != nil {
		return nil, err
	}
	return data, nil
}

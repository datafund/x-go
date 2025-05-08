package twitter

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	twitterscraper "github.com/imperatrona/twitter-scraper"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

// mockScraper implements the Scraper interface for testing
type mockScraper struct {
	*twitterscraper.Scraper
	isLoggedIn bool
}

func (m *mockScraper) IsLoggedIn() bool {
	return m.isLoggedIn
}

func (m *mockScraper) SetCookies(cookies []*http.Cookie) {
	// Mock implementation
}

func (m *mockScraper) GetProfile(ctx context.Context, username string) (*twitterscraper.Profile, error) {
	return &twitterscraper.Profile{}, nil
}

func (m *mockScraper) GetTweets(ctx context.Context, username string, maxTweetsNb int) <-chan *twitterscraper.TweetResult {
	ch := make(chan *twitterscraper.TweetResult)
	close(ch)
	return ch
}

func (m *mockScraper) GetTweet(ctx context.Context, id string) (*twitterscraper.Tweet, error) {
	return &twitterscraper.Tweet{}, nil
}

func (m *mockScraper) SearchTweets(ctx context.Context, query string, maxTweetsNb int) <-chan *twitterscraper.TweetResult {
	ch := make(chan *twitterscraper.TweetResult)
	close(ch)
	return ch
}

func (m *mockScraper) Tweet(ctx context.Context, text string) (*twitterscraper.Tweet, error) {
	return &twitterscraper.Tweet{}, nil
}

func (m *mockScraper) LikeTweet(ctx context.Context, id string) error {
	return nil
}

func (m *mockScraper) UnlikeTweet(ctx context.Context, id string) error {
	return nil
}

func (m *mockScraper) CreateRetweet(ctx context.Context, id string) error {
	return nil
}

func (m *mockScraper) CreateScheduledTweet(ctx context.Context, text string, scheduleTime string) error {
	return nil
}

func TestNewAgent(t *testing.T) {
	agent := newMockAgent()
	assert.NotNil(t, agent)
	assert.NotNil(t, agent.scraper)
}

func TestGetTools(t *testing.T) {
	agent := newMockAgent()
	tools := agent.GetTools()

	// Without login, only basic tools should be available
	assert.Equal(t, 3, len(tools), "Without login, only 3 basic tools should be available")

	// Map of expected tool names and their required parameters
	expectedBasicTools := map[string]struct {
		required   []string
		readOnly   bool
		openWorld  bool
		hasHandler bool
	}{
		"get_user_tweets": {
			required:   []string{"username"},
			readOnly:   true,
			openWorld:  true,
			hasHandler: true,
		},
		"get_profile": {
			required:   []string{"username"},
			readOnly:   true,
			openWorld:  true,
			hasHandler: true,
		},
		"get_tweet": {
			required:   []string{"tweet_id"},
			readOnly:   true,
			openWorld:  true,
			hasHandler: true,
		},
	}

	for _, tool := range tools {
		// Check if tool exists in expected tools
		expected, exists := expectedBasicTools[tool.Tool.Name]
		assert.True(t, exists, "Unexpected tool: %s", tool.Tool.Name)

		// Check required parameters
		assert.Equal(t, expected.required, tool.Tool.InputSchema.Required, "Incorrect required parameters for %s", tool.Tool.Name)

		// Check annotations
		assert.Equal(t, expected.readOnly, tool.Tool.Annotations.ReadOnlyHint, "Incorrect ReadOnlyHint for %s", tool.Tool.Name)
		assert.Equal(t, expected.openWorld, tool.Tool.Annotations.OpenWorldHint, "Incorrect OpenWorldHint for %s", tool.Tool.Name)
		assert.NotEmpty(t, tool.Tool.Annotations.Title, "Missing Title for %s", tool.Tool.Name)

		// Check handler
		assert.NotNil(t, tool.Handler, "Missing handler for %s", tool.Tool.Name)
	}

	// Now test with login
	agent.scraper.(*mockScraper).isLoggedIn = true
	tools = agent.GetTools()

	// With login, all tools should be available
	assert.Equal(t, 8, len(tools), "With login, all tools should be available")

	// Map of expected tool names and their required parameters
	expectedAllTools := map[string]struct {
		required   []string
		readOnly   bool
		openWorld  bool
		hasHandler bool
	}{
		"get_user_tweets": {
			required:   []string{"username"},
			readOnly:   true,
			openWorld:  true,
			hasHandler: true,
		},
		"get_profile": {
			required:   []string{"username"},
			readOnly:   true,
			openWorld:  true,
			hasHandler: true,
		},
		"get_tweet": {
			required:   []string{"tweet_id"},
			readOnly:   true,
			openWorld:  true,
			hasHandler: true,
		},
		"search_tweets": {
			required:   []string{"query"},
			readOnly:   true,
			openWorld:  true,
			hasHandler: true,
		},
		"create_tweet": {
			required:   []string{"text"},
			readOnly:   false,
			openWorld:  false,
			hasHandler: true,
		},
		"like_tweet": {
			required:   []string{"tweet_id"},
			readOnly:   false,
			openWorld:  false,
			hasHandler: true,
		},
		"unlike_tweet": {
			required:   []string{"tweet_id"},
			readOnly:   false,
			openWorld:  false,
			hasHandler: true,
		},
		"retweet": {
			required:   []string{"tweet_id"},
			readOnly:   false,
			openWorld:  false,
			hasHandler: true,
		},
	}

	for _, tool := range tools {
		// Check if tool exists in expected tools
		expected, exists := expectedAllTools[tool.Tool.Name]
		assert.True(t, exists, "Unexpected tool: %s", tool.Tool.Name)

		// Check required parameters
		assert.Equal(t, expected.required, tool.Tool.InputSchema.Required, "Incorrect required parameters for %s", tool.Tool.Name)

		// Check annotations
		assert.Equal(t, expected.readOnly, tool.Tool.Annotations.ReadOnlyHint, "Incorrect ReadOnlyHint for %s", tool.Tool.Name)
		assert.Equal(t, expected.openWorld, tool.Tool.Annotations.OpenWorldHint, "Incorrect OpenWorldHint for %s", tool.Tool.Name)
		assert.NotEmpty(t, tool.Tool.Annotations.Title, "Missing Title for %s", tool.Tool.Name)

		// Check handler
		assert.NotNil(t, tool.Handler, "Missing handler for %s", tool.Tool.Name)
	}
}

func TestHandleGetUserTweetsValidation(t *testing.T) {
	agent := newMockAgent()
	ctx := context.Background()

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantError   bool
		errorString string
	}{
		{
			name:        "missing username",
			params:      map[string]interface{}{},
			wantError:   true,
			errorString: "username parameter is required",
		},
		{
			name:        "empty username",
			params:      map[string]interface{}{"username": ""},
			wantError:   true,
			errorString: "username parameter is required",
		},
		{
			name:   "valid username with limit",
			params: map[string]interface{}{"username": "testuser", "limit": float64(10)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      "get_user_tweets",
					Arguments: tt.params,
				},
			}

			result, err := agent.handleGetUserTweets(ctx, request)
			assert.NoError(t, err)

			if tt.wantError {
				assert.True(t, result.IsError)
				assert.Equal(t, tt.errorString, result.Content[0].(*mcp.TextContent).Text)
			} else {
				// Skip validation if there's an error from the Twitter API
				if result.IsError {
					t.Skip("Skipping due to Twitter API error")
				}
			}
		})
	}
}

func TestHandleGetProfileValidation(t *testing.T) {
	agent := newMockAgent()
	ctx := context.Background()

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantError   bool
		errorString string
	}{
		{
			name:        "missing username",
			params:      map[string]interface{}{},
			wantError:   true,
			errorString: "username parameter is required",
		},
		{
			name:        "empty username",
			params:      map[string]interface{}{"username": ""},
			wantError:   true,
			errorString: "username parameter is required",
		},
		{
			name:   "valid username",
			params: map[string]interface{}{"username": "testuser"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      "get_profile",
					Arguments: tt.params,
				},
			}

			result, err := agent.handleGetProfile(ctx, request)
			assert.NoError(t, err)

			if tt.wantError {
				assert.True(t, result.IsError)
				assert.Equal(t, tt.errorString, result.Content[0].(*mcp.TextContent).Text)
			} else {
				// Skip validation if there's an error from the Twitter API
				if result.IsError {
					t.Skip("Skipping due to Twitter API error")
				}
			}
		})
	}
}

func TestHandleGetTweetValidation(t *testing.T) {
	agent := newMockAgent()
	ctx := context.Background()

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantError   bool
		errorString string
	}{
		{
			name:        "missing tweet_id",
			params:      map[string]interface{}{},
			wantError:   true,
			errorString: "tweet_id parameter is required",
		},
		{
			name:        "empty tweet_id",
			params:      map[string]interface{}{"tweet_id": ""},
			wantError:   true,
			errorString: "tweet_id parameter is required",
		},
		{
			name:   "valid tweet_id",
			params: map[string]interface{}{"tweet_id": "123456789"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      "get_tweet",
					Arguments: tt.params,
				},
			}

			result, err := agent.handleGetTweet(ctx, request)
			assert.NoError(t, err)

			if tt.wantError {
				assert.True(t, result.IsError)
				assert.Equal(t, tt.errorString, result.Content[0].(*mcp.TextContent).Text)
			} else {
				// Skip validation if there's an error from the Twitter API
				if result.IsError {
					t.Skip("Skipping due to Twitter API error")
				}
			}
		})
	}
}

func TestHandleSearchTweetsValidation(t *testing.T) {
	agent := newMockAgent()
	ctx := context.Background()

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantError   bool
		errorString string
	}{
		{
			name:        "missing query",
			params:      map[string]interface{}{},
			wantError:   true,
			errorString: "query parameter is required",
		},
		{
			name:        "empty query",
			params:      map[string]interface{}{"query": ""},
			wantError:   true,
			errorString: "query parameter is required",
		},
		{
			name:   "valid query with limit",
			params: map[string]interface{}{"query": "test", "limit": float64(10)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      "search_tweets",
					Arguments: tt.params,
				},
			}

			result, err := agent.handleSearchTweets(ctx, request)
			assert.NoError(t, err)

			if tt.wantError {
				assert.True(t, result.IsError)
				assert.Equal(t, tt.errorString, result.Content[0].(*mcp.TextContent).Text)
			} else {
				// Skip validation if there's an error from the Twitter API
				if result.IsError {
					t.Skip("Skipping due to Twitter API error")
				}
			}
		})
	}
}

func TestJSONResponseFormat(t *testing.T) {
	agent := newMockAgent()
	ctx := context.Background()

	// Test JSON response format for each handler
	tests := []struct {
		name     string
		request  mcp.CallToolRequest
		handler  func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		validate func(t *testing.T, jsonStr string)
	}{
		{
			name: "get_profile JSON format",
			request: mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      "get_profile",
					Arguments: map[string]interface{}{"username": "testuser"},
				},
			},
			handler: agent.handleGetProfile,
			validate: func(t *testing.T, jsonStr string) {
				var profile map[string]interface{}
				err := json.Unmarshal([]byte(jsonStr), &profile)
				assert.NoError(t, err)

				// Check required fields
				requiredFields := []string{
					"username", "name", "bio", "followers", "following",
					"tweets", "likes", "joined", "verified", "private",
					"avatar_url", "banner_url", "location", "website",
					"pinned_tweet",
				}
				for _, field := range requiredFields {
					_, exists := profile[field]
					assert.True(t, exists, "Missing field: %s", field)
				}
			},
		},
		{
			name: "get_tweet JSON format",
			request: mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      "get_tweet",
					Arguments: map[string]interface{}{"tweet_id": "123456789"},
				},
			},
			handler: agent.handleGetTweet,
			validate: func(t *testing.T, jsonStr string) {
				var tweet map[string]interface{}
				err := json.Unmarshal([]byte(jsonStr), &tweet)
				assert.NoError(t, err)

				// Check required fields
				requiredFields := []string{
					"id", "text", "likes", "retweets", "replies",
					"timestamp", "author",
				}
				for _, field := range requiredFields {
					_, exists := tweet[field]
					assert.True(t, exists, "Missing field: %s", field)
				}

				// Check author fields
				author, ok := tweet["author"].(map[string]interface{})
				assert.True(t, ok)
				authorFields := []string{"username", "name", "verified"}
				for _, field := range authorFields {
					_, exists := author[field]
					assert.True(t, exists, "Missing author field: %s", field)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.handler(ctx, tt.request)
			if err != nil {
				// Skip validation if there's an error from the Twitter API
				t.Skip("Skipping due to Twitter API error")
				return
			}

			if result.IsError {
				t.Skip("Skipping due to Twitter API error")
				return
			}

			content := result.Content[0].(*mcp.TextContent)
			tt.validate(t, content.Text)
		})
	}
}

func TestHandleLoginWithCookies(t *testing.T) {
	t.Skip("Login with cookies is now handled in main.go")
}

func TestHandleCreateTweet(t *testing.T) {
	agent := newMockAgent()
	ctx := context.Background()

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantError   bool
		errorString string
	}{
		{
			name:        "missing text",
			params:      map[string]interface{}{},
			wantError:   true,
			errorString: "tweet text is required",
		},
		{
			name:        "empty text",
			params:      map[string]interface{}{"text": ""},
			wantError:   true,
			errorString: "tweet text is required",
		},
		{
			name: "valid text",
			params: map[string]interface{}{
				"text": "Test tweet",
			},
		},
		{
			name: "invalid schedule time",
			params: map[string]interface{}{
				"text":          "Test tweet",
				"schedule_time": "invalid-time",
			},
			wantError:   true,
			errorString: "invalid schedule time format",
		},
		{
			name: "valid schedule time",
			params: map[string]interface{}{
				"text":          "Test tweet",
				"schedule_time": time.Now().Add(time.Hour).Format(time.RFC3339),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      "create_tweet",
					Arguments: tt.params,
				},
			}

			result, err := agent.handleCreateTweet(ctx, request)
			assert.NoError(t, err)

			if tt.wantError {
				assert.True(t, result.IsError)
				assert.Contains(t, result.Content[0].(*mcp.TextContent).Text, tt.errorString)
			}
		})
	}
}

func TestHandleLikeUnlikeTweet(t *testing.T) {
	agent := newMockAgent()
	ctx := context.Background()

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantError   bool
		errorString string
		handler     func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
	}{
		{
			name:        "missing tweet_id for like",
			params:      map[string]interface{}{},
			wantError:   true,
			errorString: "tweet_id is required",
			handler:     agent.handleLikeTweet,
		},
		{
			name:        "empty tweet_id for like",
			params:      map[string]interface{}{"tweet_id": ""},
			wantError:   true,
			errorString: "tweet_id is required",
			handler:     agent.handleLikeTweet,
		},
		{
			name:        "missing tweet_id for unlike",
			params:      map[string]interface{}{},
			wantError:   true,
			errorString: "tweet_id is required",
			handler:     agent.handleUnlikeTweet,
		},
		{
			name:        "empty tweet_id for unlike",
			params:      map[string]interface{}{"tweet_id": ""},
			wantError:   true,
			errorString: "tweet_id is required",
			handler:     agent.handleUnlikeTweet,
		},
		{
			name: "valid tweet_id for like",
			params: map[string]interface{}{
				"tweet_id": "123456789",
			},
			handler: agent.handleLikeTweet,
		},
		{
			name: "valid tweet_id for unlike",
			params: map[string]interface{}{
				"tweet_id": "123456789",
			},
			handler: agent.handleUnlikeTweet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      "like_tweet",
					Arguments: tt.params,
				},
			}

			result, err := tt.handler(ctx, request)
			assert.NoError(t, err)

			if tt.wantError {
				assert.True(t, result.IsError)
				assert.Equal(t, tt.errorString, result.Content[0].(*mcp.TextContent).Text)
			}
		})
	}
}

func TestHandleRetweet(t *testing.T) {
	agent := newMockAgent()
	ctx := context.Background()

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantError   bool
		errorString string
	}{
		{
			name:        "missing tweet_id",
			params:      map[string]interface{}{},
			wantError:   true,
			errorString: "tweet_id is required",
		},
		{
			name:        "empty tweet_id",
			params:      map[string]interface{}{"tweet_id": ""},
			wantError:   true,
			errorString: "tweet_id is required",
		},
		{
			name: "valid tweet_id",
			params: map[string]interface{}{
				"tweet_id": "123456789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      "retweet",
					Arguments: tt.params,
				},
			}

			result, err := agent.handleRetweet(ctx, request)
			assert.NoError(t, err)

			if tt.wantError {
				assert.True(t, result.IsError)
				assert.Equal(t, tt.errorString, result.Content[0].(*mcp.TextContent).Text)
			}
		})
	}
}

func TestLoginRequiredTools(t *testing.T) {
	agent := newMockAgent()
	ctx := context.Background()

	// Test each tool that requires login
	tests := []struct {
		name    string
		handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		params  map[string]interface{}
	}{
		{
			name:    "search_tweets",
			handler: agent.handleSearchTweets,
			params: map[string]interface{}{
				"query": "test",
			},
		},
		{
			name:    "create_tweet",
			handler: agent.handleCreateTweet,
			params: map[string]interface{}{
				"text": "test",
			},
		},
		{
			name:    "like_tweet",
			handler: agent.handleLikeTweet,
			params: map[string]interface{}{
				"tweet_id": "123",
			},
		},
		{
			name:    "unlike_tweet",
			handler: agent.handleUnlikeTweet,
			params: map[string]interface{}{
				"tweet_id": "123",
			},
		},
		{
			name:    "retweet",
			handler: agent.handleRetweet,
			params: map[string]interface{}{
				"tweet_id": "123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" without login", func(t *testing.T) {
			// Reset login state before each test
			agent.scraper.(*mockScraper).isLoggedIn = false
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      tt.name,
					Arguments: tt.params,
				},
			}

			result, err := tt.handler(ctx, request)
			assert.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Equal(t, "This tool requires login. Please provide Twitter cookies to use this tool.", result.Content[0].(*mcp.TextContent).Text)
		})

		t.Run(tt.name+" with login", func(t *testing.T) {
			// Set login state to true for this test
			agent.scraper.(*mockScraper).isLoggedIn = true
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      tt.name,
					Arguments: tt.params,
				},
			}

			result, err := tt.handler(ctx, request)
			assert.NoError(t, err)
			if result.IsError {
				t.Skip("Skipping due to Twitter API error")
			}
		})
	}
}

func newMockAgent() *Agent {
	return &Agent{
		scraper: &mockScraper{
			Scraper:    twitterscraper.New(),
			isLoggedIn: false,
		},
		limiter: newRateLimiter(),
	}
}

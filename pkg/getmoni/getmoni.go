package getmoni

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"
)

// Logger is a simple logger interface
type Logger interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
	Warning(format string, args ...interface{})
}

// DefaultLogger implements the Logger interface using the standard log package
type DefaultLogger struct {
	*log.Logger
}

// NewDefaultLogger creates a new default logger
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{
		Logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// Info logs an info message
func (l *DefaultLogger) Info(format string, args ...interface{}) {
	l.Printf("[INFO] "+format, args...)
}

// Error logs an error message
func (l *DefaultLogger) Error(format string, args ...interface{}) {
	l.Printf("[ERROR] "+format, args...)
}

// Warning logs a warning message
func (l *DefaultLogger) Warning(format string, args ...interface{}) {
	l.Printf("[WARNING] "+format, args...)
}

// GetMoni represents the GetMoni API client
type GetMoni struct {
	baseURL string
	apiKey  string
	client  *http.Client
	logger  Logger
}

// Link represents a social media link in the user's profile
type Link struct {
	URL     string `json:"url"`
	LogoURL string `json:"logoUrl"`
	Type    string `json:"type"`
	Name    string `json:"name"`
}

// UserMeta represents the metadata of a Twitter user
type UserMeta struct {
	TwitterUserID    int64  `json:"twitterUserId"`
	Username         string `json:"username"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	ProfileImageURL  string `json:"profileImageUrl"`
	ProfileBannerURL string `json:"profileBannerUrl"`
	TwitterCreatedAt int64  `json:"twitterCreatedAt"`
	TweetCount       int    `json:"tweetCount"`
	FollowersCount   int    `json:"followersCount"`
	Links            []Link `json:"links"`
}

// SmartFollowerItem represents a single item in the smart followers response
type SmartFollowerItem struct {
	Meta UserMeta `json:"meta"`
}

// SmartFollowersResponse represents the response from GetMoni's smart followers endpoint
type SmartFollowersResponse struct {
	Items      []SmartFollowerItem `json:"items"`
	TotalCount int                 `json:"totalCount"`
}

// NewGetMoni creates a new GetMoni client
func NewGetMoni(apiKey string) *GetMoni {
	if apiKey == "" {
		apiKey = os.Getenv("GETMONI_API_KEY")
	}

	client := &GetMoni{
		baseURL: "https://api.discover.getmoni.io/api/v2",
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 30 * time.Second},
		logger:  NewDefaultLogger(),
	}

	// Check server status on init
	status, err := client.makeRequest("GET", "/status/server/", nil, nil)
	if err != nil {
		client.logger.Error("Failed to check GetMoni server status: %v", err)
	} else {
		client.logger.Info("GetMoni server status: %v", status)
	}

	return client
}

// makeRequest makes an HTTP request to the GetMoni API with exponential backoff retry logic
func (g *GetMoni) makeRequest(method, endpoint string, params map[string]string, data interface{}) (map[string]interface{}, error) {
	if g.apiKey == "" {
		g.logger.Warning("GetMoni API key not available, skipping API call")
		return map[string]interface{}{"error": "API key not available"}, nil
	}

	maxRetries := 10
	baseWait := 1.0

	for retryCount := 0; retryCount < maxRetries; retryCount++ {
		url := g.baseURL + endpoint
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %v", err)
		}

		// Add headers
		req.Header.Set("Api-Key", g.apiKey)
		req.Header.Set("accept", "application/json")

		// Add query parameters
		q := req.URL.Query()
		for k, v := range params {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()

		// Make request
		resp, err := g.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error making request: %v", err)
		}
		defer resp.Body.Close()

		// Handle rate limiting
		if resp.StatusCode == http.StatusTooManyRequests {
			waitTime := baseWait * math.Pow(2, float64(retryCount))
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if retryAfterFloat, err := time.ParseDuration(retryAfter + "s"); err == nil {
					waitTime = float64(retryAfterFloat.Seconds())
				}
			}

			g.logger.Warning("Rate limited on %s. Retry attempt %d/%d. Waiting %.2f seconds...",
				endpoint, retryCount+1, maxRetries, waitTime)
			time.Sleep(time.Duration(waitTime * float64(time.Second)))
			continue
		}

		// Parse response using a more flexible approach
		var result map[string]interface{}
		var rawResult interface{}

		if err := json.NewDecoder(resp.Body).Decode(&rawResult); err != nil {
			return nil, fmt.Errorf("error decoding response: %v", err)
		}

		// Handle different response types
		switch v := rawResult.(type) {
		case map[string]interface{}:
			result = v
		case float64, int, string, bool:
			// Wrap primitive types in a map
			result = map[string]interface{}{
				"value": v,
			}
		default:
			result = map[string]interface{}{
				"value": v,
			}
		}

		return result, nil
	}

	return nil, fmt.Errorf("max retries (%d) reached", maxRetries)
}

// GetSmartFollowers gets smart followers for a Twitter username
func (g *GetMoni) GetSmartFollowers(username string, limit, offset int, orderBy, orderByDirection string) (*SmartFollowersResponse, error) {
	params := map[string]string{
		"limit":            fmt.Sprintf("%d", limit),
		"offset":           fmt.Sprintf("%d", offset),
		"orderBy":          orderBy,
		"orderByDirection": orderByDirection,
	}

	result, err := g.makeRequest("GET", fmt.Sprintf("/twitters/%s/smart_followers/meta", username), params, nil)
	if err != nil {
		return nil, err
	}

	// Convert the result to JSON and then to our struct
	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling result: %v", err)
	}

	var response SmartFollowersResponse
	if err := json.Unmarshal(jsonData, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response, nil
}

// GetSmartMentions gets smart mentions for a Twitter username
func (g *GetMoni) GetSmartMentions(username string, fromDate, toDate string, limit int) (map[string]interface{}, error) {
	params := map[string]string{
		"limit": fmt.Sprintf("%d", limit),
	}

	if fromDate != "" {
		params["fromDate"] = fromDate
	}
	if toDate != "" {
		params["toDate"] = toDate
	}

	return g.makeRequest("GET", fmt.Sprintf("/twitters/%s/feed/smart_mentions", username), params, nil)
}

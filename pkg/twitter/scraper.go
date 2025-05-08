package twitter

import (
	"context"
	"net/http"

	twitterscraper "github.com/imperatrona/twitter-scraper"
)

// scraperWrapper wraps the twitter-scraper to match our interface
type scraperWrapper struct {
	*twitterscraper.Scraper
}

func newScraperWrapper() *scraperWrapper {
	return &scraperWrapper{
		Scraper: twitterscraper.New(),
	}
}

func (s *scraperWrapper) IsLoggedIn() bool {
	return s.Scraper.IsLoggedIn()
}

func (s *scraperWrapper) GetProfile(ctx context.Context, username string) (*twitterscraper.Profile, error) {
	profile, err := s.Scraper.GetProfile(username)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *scraperWrapper) GetTweets(ctx context.Context, username string, maxTweetsNb int) <-chan *twitterscraper.TweetResult {
	return s.Scraper.GetTweets(ctx, username, maxTweetsNb)
}

func (s *scraperWrapper) GetTweet(ctx context.Context, id string) (*twitterscraper.Tweet, error) {
	tweet, err := s.Scraper.GetTweet(id)
	if err != nil {
		return nil, err
	}
	return tweet, nil
}

func (s *scraperWrapper) SearchTweets(ctx context.Context, query string, maxTweetsNb int) <-chan *twitterscraper.TweetResult {
	return s.Scraper.SearchTweets(ctx, query, maxTweetsNb)
}

func (s *scraperWrapper) Tweet(ctx context.Context, text string) (*twitterscraper.Tweet, error) {
	tweet := twitterscraper.NewTweet{
		Text: text,
	}
	result, err := s.Scraper.CreateTweet(tweet)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *scraperWrapper) Follow(ctx context.Context, id string) error {
	return s.Scraper.Follow(id)
}

func (s *scraperWrapper) Unfollow(ctx context.Context, id string) error {
	return s.Scraper.Unfollow(id)
}

func (s *scraperWrapper) LikeTweet(ctx context.Context, id string) error {
	return s.Scraper.LikeTweet(id)
}

func (s *scraperWrapper) UnlikeTweet(ctx context.Context, id string) error {
	return s.Scraper.UnlikeTweet(id)
}

func (s *scraperWrapper) CreateRetweet(ctx context.Context, id string) error {
	_, err := s.Scraper.CreateRetweet(id)
	return err
}

func (s *scraperWrapper) CreateScheduledTweet(ctx context.Context, text string, scheduleTime string) error {
	// Note: The twitter-scraper package doesn't support scheduled tweets directly
	// We'll need to implement this feature differently or use a different package
	return nil
}

func (s *scraperWrapper) GetCookies() []*http.Cookie {
	return s.Scraper.GetCookies()
}

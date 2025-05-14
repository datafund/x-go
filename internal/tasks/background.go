package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/asabya/x-go/pkg/twitter"
)

type Profile struct {
	ID                   int64
	UserID               string
	Username             string
	Name                 string
	Biography            string
	Avatar               string
	Banner               string
	Birthday             string
	Location             string
	URL                  string
	Website              string
	Joined               time.Time
	TweetsCount          int
	LikesCount           int
	MediaCount           int
	FollowersCount       int
	FollowingCount       int
	FriendsCount         int
	NormalFollowersCount int
	FastFollowersCount   int
	ListedCount          int
	IsVerified           bool
	IsPrivate            bool
	IsBlueVerified       bool
	CanHighlightTweets   bool
	HasGraduatedAccess   bool
	FollowedBy           bool
	Following            bool
	Sensitive            bool
	ProfileImageShape    string
}

type Tweet struct {
	ID                string
	UserID            string
	Username          string
	Name              string
	Text              string
	HTML              string
	TimeParsed        time.Time
	Timestamp         int64
	PermanentURL      string
	Likes             int
	Replies           int
	Retweets          int
	Views             int
	IsPin             bool
	IsReply           bool
	IsQuoted          bool
	IsRetweet         bool
	IsSelfThread      bool
	SensitiveContent  bool
	RetweetedStatusID string
	QuotedStatusID    string
	InReplyToStatusID string
	Place             string
}

// StartProfileUpdates starts a goroutine that updates user profiles periodically
func StartProfileUpdates(db *sql.DB, agentManager *twitter.AgentManager, logger *log.Logger) {
	go func() {
		for {
			rows, err := db.Query("SELECT username FROM users")
			if err != nil {
				logger.Printf("Error querying users: %v", err)
				time.Sleep(10 * time.Second)
				continue
			}
			defer rows.Close()

			for rows.Next() {
				var username string
				if err := rows.Scan(&username); err != nil {
					logger.Printf("Error scanning username: %v", err)
					continue
				}

				profileData, _, err := agentManager.GetProfile(context.Background(), username)
				if err != nil {
					logger.Printf("Error getting profile for %s: %v", username, err)
					continue
				}

				// Convert interface{} to Profile struct
				profileBytes, err := json.Marshal(profileData)
				if err != nil {
					logger.Printf("Error marshaling profile data: %v", err)
					continue
				}

				var profile Profile
				if err := json.Unmarshal(profileBytes, &profile); err != nil {
					logger.Printf("Error unmarshaling profile data: %v", err)
					continue
				}

				// Update user profile in database
				_, err = db.Exec(`
					UPDATE users SET 
						user_id = $1, name = $2, biography = $3, avatar = $4, banner = $5,
						location = $6, url = $7, website = $8, joined = $9,
						tweets_count = $10, likes_count = $11, media_count = $12,
						followers_count = $13, following_count = $14, friends_count = $15,
						normal_followers_count = $16, fast_followers_count = $17, listed_count = $18,
						is_verified = $19, is_private = $20, is_blue_verified = $21,
						can_highlight_tweets = $22, has_graduated_access = $23,
						followed_by = $24, following = $25, sensitive = $26,
						profile_image_shape = $27
					WHERE username = $28`,
					profile.UserID, profile.Name, profile.Biography, profile.Avatar, profile.Banner,
					profile.Location, profile.URL, profile.Website, profile.Joined,
					profile.TweetsCount, profile.LikesCount, profile.MediaCount,
					profile.FollowersCount, profile.FollowingCount, profile.FriendsCount,
					profile.NormalFollowersCount, profile.FastFollowersCount, profile.ListedCount,
					profile.IsVerified, profile.IsPrivate, profile.IsBlueVerified,
					profile.CanHighlightTweets, profile.HasGraduatedAccess,
					profile.FollowedBy, profile.Following, profile.Sensitive,
					profile.ProfileImageShape, username)

				if err != nil {
					logger.Printf("Error updating profile for %s: %v", username, err)
				}

				time.Sleep(10 * time.Second)
			}
		}
	}()
}

// StartTweetUpdates starts a goroutine that updates user tweets periodically
func StartTweetUpdates(db *sql.DB, agentManager *twitter.AgentManager, logger *log.Logger) {
	go func() {
		for {
			rows, err := db.Query("SELECT username, id FROM users")
			if err != nil {
				logger.Printf("Error querying users: %v", err)
				time.Sleep(time.Hour)
				continue
			}
			defer rows.Close()

			for rows.Next() {
				var username string
				var userID string
				if err := rows.Scan(&username, &userID); err != nil {
					logger.Printf("Error scanning user data: %v", err)
					continue
				}

				tweetsData, _, err := agentManager.GetUserTweets(context.Background(), username, 20, false)
				if err != nil {
					logger.Printf("Error getting tweets for %s: %v", username, err)
					continue
				}

				// Convert interface{} to []Tweet
				tweetsBytes, err := json.Marshal(tweetsData)
				if err != nil {
					logger.Printf("Error marshaling tweets data: %v", err)
					continue
				}

				var tweets []Tweet
				if err := json.Unmarshal(tweetsBytes, &tweets); err != nil {
					logger.Printf("Error unmarshaling tweets data: %v", err)
					continue
				}

				for _, tweet := range tweets {
					// Insert tweet if it doesn't exist
					_, err = db.Exec(`
						INSERT INTO tweets (
							id, user_id, tweeter_user_id, username, name, text, html,
							time_parsed, timestamp, permanent_url, likes, replies,
							retweets, views, is_pin, is_reply, is_quoted, is_retweet,
							is_self_thread, sensitive_content, retweeted_status_id,
							quoted_status_id, in_reply_to_status_id, place
						) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
						ON CONFLICT (id) DO UPDATE SET
							likes = EXCLUDED.likes,
							replies = EXCLUDED.replies,
							retweets = EXCLUDED.retweets,
							views = EXCLUDED.views`,
						tweet.ID, userID, tweet.UserID, tweet.Username, tweet.Name, tweet.Text, tweet.HTML,
						tweet.TimeParsed, tweet.Timestamp, tweet.PermanentURL, tweet.Likes, tweet.Replies,
						tweet.Retweets, tweet.Views, tweet.IsPin, tweet.IsReply, tweet.IsQuoted, tweet.IsRetweet,
						tweet.IsSelfThread, tweet.SensitiveContent, tweet.RetweetedStatusID,
						tweet.QuotedStatusID, tweet.InReplyToStatusID, tweet.Place)

					if err != nil {
						logger.Printf("Error inserting/updating tweet: %v", err)
					}
				}
			}

			time.Sleep(6 * time.Hour)
		}
	}()
}

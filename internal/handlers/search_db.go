package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type SearchResponse struct {
	Tweets []Tweet `json:"tweets"`
}

type Tweet struct {
	ID                string `json:"id"`
	UserID            string `json:"user_id"`
	TweeterUserID     string `json:"tweeter_user_id"`
	Username          string `json:"username"`
	Name              string `json:"name"`
	Text              string `json:"text"`
	HTML              string `json:"html"`
	TimeParsed        string `json:"time_parsed"`
	Timestamp         int64  `json:"timestamp"`
	PermanentURL      string `json:"permanent_url"`
	Likes             int    `json:"likes"`
	Replies           int    `json:"replies"`
	Retweets          int    `json:"retweets"`
	Views             int    `json:"views"`
	IsPin             bool   `json:"is_pin"`
	IsReply           bool   `json:"is_reply"`
	IsQuoted          bool   `json:"is_quoted"`
	IsRetweet         bool   `json:"is_retweet"`
	IsSelfThread      bool   `json:"is_self_thread"`
	SensitiveContent  bool   `json:"sensitive_content"`
	RetweetedStatusID string `json:"retweeted_status_id"`
	QuotedStatusID    string `json:"quoted_status_id"`
	InReplyToStatusID string `json:"in_reply_to_status_id"`
	Place             string `json:"place"`
	// User fields
	UserIsVerified     bool `json:"user_is_verified"`
	UserIsPrivate      bool `json:"user_is_private"`
	UserIsBlueVerified bool `json:"user_is_blue_verified"`
	UserFollowingCount int  `json:"user_following_count"`
	UserFollowersCount int  `json:"user_followers_count"`
	UserLikesCount     int  `json:"user_likes_count"`
	UserTweetsCount    int  `json:"user_tweets_count"`
}

// HandleSearchTweetsInDB handles searching tweets in the database
func HandleSearchTweetsInDB(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
			return
		}

		// Get sorting parameters
		sortBy := r.URL.Query().Get("sort_by")
		if sortBy == "" {
			sortBy = "timestamp" // default sort by timestamp
		}

		// Validate sort_by parameter
		validSortFields := map[string]bool{
			"timestamp": true,
			"likes":     true,
			"views":     true,
		}
		if !validSortFields[sortBy] {
			http.Error(w, "Invalid sort_by parameter. Must be one of: timestamp, likes, views", http.StatusBadRequest)
			return
		}

		// Get limit parameter
		limit := 50 // default limit
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			parsedLimit, err := strconv.Atoi(limitStr)
			if err != nil || parsedLimit <= 0 {
				http.Error(w, "Invalid limit parameter. Must be a positive integer", http.StatusBadRequest)
				return
			}
			limit = parsedLimit
		}

		// Build the query with user join
		sqlQuery := `
			SELECT 
				t.id, t.user_id, t.tweeter_user_id, t.username, t.name, t.text, t.html,
				t.time_parsed, t.timestamp, t.permanent_url, t.likes, t.replies,
				t.retweets, t.views, t.is_pin, t.is_reply, t.is_quoted, t.is_retweet,
				t.is_self_thread, t.sensitive_content, t.retweeted_status_id,
				t.quoted_status_id, t.in_reply_to_status_id, t.place,
				u.is_verified as user_is_verified, u.is_private as user_is_private,
				u.is_blue_verified as user_is_blue_verified,
				u.following_count as user_following_count,
				u.followers_count as user_followers_count,
				u.likes_count as user_likes_count,
				u.tweets_count as user_tweets_count
			FROM tweets t
			LEFT JOIN users u ON t.user_id = u.id
			WHERE t.text ILIKE $1
			ORDER BY t.` + sortBy + ` DESC
			LIMIT $2`

		rows, err := db.Query(sqlQuery, "%"+query+"%", limit)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error executing query: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var tweets []Tweet
		for rows.Next() {
			var t Tweet
			// Temporary variables for handling NULL values
			var userIsVerified, userIsPrivate, userIsBlueVerified sql.NullBool
			var userFollowingCount, userFollowersCount, userLikesCount, userTweetsCount sql.NullInt64

			err := rows.Scan(
				&t.ID, &t.UserID, &t.TweeterUserID, &t.Username, &t.Name, &t.Text, &t.HTML,
				&t.TimeParsed, &t.Timestamp, &t.PermanentURL, &t.Likes, &t.Replies,
				&t.Retweets, &t.Views, &t.IsPin, &t.IsReply, &t.IsQuoted, &t.IsRetweet,
				&t.IsSelfThread, &t.SensitiveContent, &t.RetweetedStatusID,
				&t.QuotedStatusID, &t.InReplyToStatusID, &t.Place,
				&userIsVerified, &userIsPrivate, &userIsBlueVerified,
				&userFollowingCount, &userFollowersCount,
				&userLikesCount, &userTweetsCount,
			)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error scanning tweet: %v", err), http.StatusInternalServerError)
				return
			}

			// Convert NULL values to appropriate defaults
			t.UserIsVerified = userIsVerified.Valid && userIsVerified.Bool
			t.UserIsPrivate = userIsPrivate.Valid && userIsPrivate.Bool
			t.UserIsBlueVerified = userIsBlueVerified.Valid && userIsBlueVerified.Bool
			t.UserFollowingCount = int(userFollowingCount.Int64)
			if !userFollowingCount.Valid {
				t.UserFollowingCount = 0
			}
			t.UserFollowersCount = int(userFollowersCount.Int64)
			if !userFollowersCount.Valid {
				t.UserFollowersCount = 0
			}
			t.UserLikesCount = int(userLikesCount.Int64)
			if !userLikesCount.Valid {
				t.UserLikesCount = 0
			}
			t.UserTweetsCount = int(userTweetsCount.Int64)
			if !userTweetsCount.Valid {
				t.UserTweetsCount = 0
			}

			tweets = append(tweets, t)
		}

		response := SearchResponse{
			Tweets: tweets,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

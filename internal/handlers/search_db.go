package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type SearchResponse struct {
	Users []User `json:"users"`
}

type User struct {
	Username           string `json:"username"`
	UserIsVerified     bool   `json:"user_is_verified"`
	UserIsPrivate      bool   `json:"user_is_private"`
	UserIsBlueVerified bool   `json:"user_is_blue_verified"`
	UserFollowingCount int    `json:"user_following_count"`
	UserFollowersCount int    `json:"user_followers_count"`
	UserLikesCount     int    `json:"user_likes_count"`
	UserTweetsCount    int    `json:"user_tweets_count"`

	Tweets []Tweet `json:"tweets"`
}

// Tweet represents the simplified tweet structure for the API response
type Tweet struct {
	Text     string `json:"text"`
	Likes    int    `json:"likes"`
	Replies  int    `json:"replies"`
	Retweets int    `json:"retweets"`
	Views    int    `json:"views"`
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

		// Build the query with user join - only select needed fields
		sqlQuery := `
			SELECT 
				t.user_id,
				t.text, t.likes, t.replies, t.retweets, t.views,
				u.is_verified, u.is_private, u.is_blue_verified,
				u.following_count, u.followers_count,
				u.likes_count, u.tweets_count, u.username
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

		// Map to store users and their tweets
		userMap := make(map[int64]*User)

		for rows.Next() {
			var userID int64
			var tweet Tweet
			// Temporary variables for handling NULL values
			var userIsVerified, userIsPrivate, userIsBlueVerified sql.NullBool
			var userFollowingCount, userFollowersCount, userLikesCount, userTweetsCount sql.NullInt64
			var userUsername sql.NullString
			err := rows.Scan(
				&userID,
				&tweet.Text, &tweet.Likes, &tweet.Replies, &tweet.Retweets, &tweet.Views,
				&userIsVerified, &userIsPrivate, &userIsBlueVerified,
				&userFollowingCount, &userFollowersCount,
				&userLikesCount, &userTweetsCount, &userUsername,
			)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error scanning tweet: %v", err), http.StatusInternalServerError)
				return
			}

			// Get or create user
			user, exists := userMap[userID]
			if !exists {
				user = &User{
					UserIsVerified:     userIsVerified.Valid && userIsVerified.Bool,
					UserIsPrivate:      userIsPrivate.Valid && userIsPrivate.Bool,
					UserIsBlueVerified: userIsBlueVerified.Valid && userIsBlueVerified.Bool,
					UserFollowingCount: int(userFollowingCount.Int64),
					UserFollowersCount: int(userFollowersCount.Int64),
					UserLikesCount:     int(userLikesCount.Int64),
					UserTweetsCount:    int(userTweetsCount.Int64),
					Username:           userUsername.String,
					Tweets:             make([]Tweet, 0),
				}
				userMap[userID] = user
			}

			user.Tweets = append(user.Tweets, tweet)
		}

		// Convert map to slice
		users := make([]User, 0, len(userMap))
		for _, user := range userMap {
			users = append(users, *user)
		}

		response := SearchResponse{
			Users: users,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

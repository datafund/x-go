package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/asabya/x-go/internal/tasks"
	"github.com/asabya/x-go/pkg/twitter"
	"github.com/gorilla/mux"
)

func LoggingMiddleware(logger *log.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a response wrapper to capture the status code and headers
			rw := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
				headers:        make(http.Header),
			}

			// Call the next handler
			next.ServeHTTP(rw, r)
			logger.Printf("%s %s status: %d", r.Method, r.URL.Path, rw.status)
		})
	}
}

// responseWriter is a wrapper for http.ResponseWriter that captures the status code and headers
type responseWriter struct {
	http.ResponseWriter
	status  int
	headers http.Header
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Header() http.Header {
	return rw.ResponseWriter.Header()
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}

func HandleGetUserTweetsWithManager(manager *twitter.AgentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		username := vars["username"]
		limit := 50

		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil {
				limit = l
			}
		}

		sortByOldest := false
		if sortStr := r.URL.Query().Get("sort_by_oldest"); sortStr == "true" {
			sortByOldest = true
		}

		result, agentUsername, err := manager.GetUserTweets(r.Context(), username, limit, sortByOldest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Agent-Username", agentUsername)
		json.NewEncoder(w).Encode(result)
	}
}

func HandleGetProfileWithManager(manager *twitter.AgentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		username := vars["username"]

		result, agentUsername, err := manager.GetProfile(r.Context(), username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Agent-Username", agentUsername)
		json.NewEncoder(w).Encode(result)
	}
}

func HandleGetTweetWithManager(manager *twitter.AgentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tweetID := vars["id"]

		result, agentUsername, err := manager.GetTweet(r.Context(), tweetID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Agent-Username", agentUsername)
		json.NewEncoder(w).Encode(result)
	}
}

func HandleSearchTweetsWithManager(manager *twitter.AgentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		limit := 50

		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil {
				limit = l
			}
		}

		result, agentUsername, err := manager.SearchTweets(r.Context(), query, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Agent-Username", agentUsername)
		json.NewEncoder(w).Encode(result)
	}
}

type CreateTweetRequest struct {
	Text         string `json:"text"`
	ScheduleTime string `json:"schedule_time,omitempty"`
}

func HandleCreateTweetWithManager(manager *twitter.AgentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateTweetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		result, agentUsername, err := manager.CreateTweet(r.Context(), req.Text, req.ScheduleTime)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Agent-Username", agentUsername)
		json.NewEncoder(w).Encode(result)
	}
}

func HandleFollowUserWithManager(manager *twitter.AgentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userID := vars["id"]

		agentUsername, err := manager.Follow(r.Context(), userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Agent-Username", agentUsername)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}

func HandleUnfollowUserWithManager(manager *twitter.AgentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userID := vars["id"]

		agentUsername, err := manager.Unfollow(r.Context(), userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Agent-Username", agentUsername)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}

func HandleLikeTweetWithManager(manager *twitter.AgentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tweetID := vars["id"]

		agentUsername, err := manager.LikeTweet(r.Context(), tweetID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Agent-Username", agentUsername)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}

func HandleUnlikeTweetWithManager(manager *twitter.AgentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tweetID := vars["id"]

		agentUsername, err := manager.UnlikeTweet(r.Context(), tweetID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Agent-Username", agentUsername)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}

func HandleRetweetWithManager(manager *twitter.AgentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tweetID := vars["id"]

		agentUsername, err := manager.Retweet(r.Context(), tweetID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Agent-Username", agentUsername)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}

func HandleGetFollowersWithManager(manager *twitter.AgentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		username := vars["username"]
		limit := 50

		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil {
				limit = l
			}
		}

		cursor := r.URL.Query().Get("cursor")

		result, agentUsername, err := manager.GetFollowers(r.Context(), username, limit, cursor)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Agent-Username", agentUsername)
		json.NewEncoder(w).Encode(result)
	}
}

func HandleAddUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req tasks.Profile
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Username == "" {
			http.Error(w, "Username is required", http.StatusBadRequest)
			return
		}

		// Insert the user into the database with all fields
		_, err := db.Exec(`
			INSERT INTO users (
				user_id, username, name, biography, avatar, banner,
				birthday, location, url, website, joined,
				tweets_count, likes_count, media_count,
				followers_count, following_count, friends_count,
				normal_followers_count, fast_followers_count, listed_count,
				is_verified, is_private, is_blue_verified,
				can_highlight_tweets, has_graduated_access,
				followed_by, following, sensitive,
				profile_image_shape
			) VALUES (
				$1, $2, $3, $4, $5, $6, NULLIF($7, '')::date, $8, $9, $10, $11,
				$12, $13, $14, $15, $16, $17, $18, $19, $20,
				$21, $22, $23, $24, $25, $26, $27, $28, $29
			)
			ON CONFLICT (username) DO NOTHING`,
			req.UserID, req.Username, req.Name, req.Biography, req.Avatar, req.Banner,
			req.Birthday, req.Location, req.URL, req.Website, req.Joined,
			req.TweetsCount, req.LikesCount, req.MediaCount,
			req.FollowersCount, req.FollowingCount, req.FriendsCount,
			req.NormalFollowersCount, req.FastFollowersCount, req.ListedCount,
			req.IsVerified, req.IsPrivate, req.IsBlueVerified,
			req.CanHighlightTweets, req.HasGraduatedAccess,
			req.FollowedBy, req.Following, req.Sensitive,
			req.ProfileImageShape)

		if err != nil {
			http.Error(w, fmt.Sprintf("Error adding user: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "User added successfully",
		})
	}
}

package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

const (
	createUsersTable = `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			user_id TEXT,
			username VARCHAR(50),
			name VARCHAR(100),
			biography TEXT,
			avatar TEXT,
			banner TEXT,
			birthday DATE,
			location VARCHAR(255),
			url TEXT,
			website TEXT,
			joined TIMESTAMP,
			
			tweets_count INT,
			likes_count INT,
			media_count INT,
			followers_count INT,
			following_count INT,
			friends_count INT,
			normal_followers_count INT,
			fast_followers_count INT,
			listed_count INT,

			is_verified BOOLEAN,
			is_private BOOLEAN,
			is_blue_verified BOOLEAN,
			can_highlight_tweets BOOLEAN,
			has_graduated_access BOOLEAN,
			followed_by BOOLEAN,
			following BOOLEAN,
			sensitive BOOLEAN,

			profile_image_shape VARCHAR(50),
			UNIQUE(username)
		);`

	createTweetsTable = `
		CREATE TABLE IF NOT EXISTS tweets (
			id TEXT PRIMARY KEY,
			user_id INTEGER,
			tweeter_user_id TEXT,
			username VARCHAR(50),
			name VARCHAR(100),
			text TEXT,
			html TEXT,
			time_parsed TIMESTAMP,
			timestamp BIGINT,
			permanent_url TEXT,
			likes INT,
			replies INT,
			retweets INT,
			views INT,
			is_pin BOOLEAN,
			is_reply BOOLEAN,
			is_quoted BOOLEAN,
			is_retweet BOOLEAN,
			is_self_thread BOOLEAN,
			sensitive_content BOOLEAN,
			retweeted_status_id TEXT,
			quoted_status_id TEXT,
			in_reply_to_status_id TEXT,
			place TEXT,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (username) REFERENCES users(username)
		);`

	createSmartUsersTable = `
		CREATE TABLE IF NOT EXISTS smart_users (
			id SERIAL PRIMARY KEY,
			user_id TEXT,
			username VARCHAR(50),
			name VARCHAR(100),
			biography TEXT,
			avatar TEXT,
			banner TEXT,
			joined BIGINT,
			tweets_count INT,
			followers_count INT,
			UNIQUE(username)
		);`

	createSmartTweetsTable = `
		CREATE TABLE IF NOT EXISTS smart_tweets (
			id TEXT PRIMARY KEY,
			user_id INTEGER,
			tweeter_user_id TEXT,
			username VARCHAR(50),
			name VARCHAR(100),
			text TEXT,
			html TEXT,
			time_parsed TIMESTAMP,
			timestamp BIGINT,
			permanent_url TEXT,
			likes INT,
			replies INT,
			retweets INT,
			views INT,
			is_pin BOOLEAN,
			is_reply BOOLEAN,
			is_quoted BOOLEAN,
			is_retweet BOOLEAN,
			is_self_thread BOOLEAN,
			sensitive_content BOOLEAN,
			retweeted_status_id TEXT,
			quoted_status_id TEXT,
			in_reply_to_status_id TEXT,
			place TEXT,
			FOREIGN KEY (user_id) REFERENCES smart_users(id),
			FOREIGN KEY (username) REFERENCES smart_users(username)
		);`
)

// InitDB initializes the database connection and creates tables
func InitDB(postgresURL string, usernames []string) (*sql.DB, error) {
	// Add sslmode=disable to the connection string if not present
	if postgresURL[len(postgresURL)-1] != '?' {
		postgresURL += "?"
	}
	if !strings.Contains(postgresURL, "sslmode=") {
		if postgresURL[len(postgresURL)-1] != '?' {
			postgresURL += "&"
		}
		postgresURL += "sslmode=disable"
	}

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to the database: %v", err)
	}

	// Create tables
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("error creating tables: %v", err)
	}

	// Insert usernames
	// if err := insertUsernames(db, usernames); err != nil {
	// 	return nil, fmt.Errorf("error inserting usernames: %v", err)
	// }

	return db, nil
}

func createTables(db *sql.DB) error {
	// Create users table
	if _, err := db.Exec(createUsersTable); err != nil {
		return fmt.Errorf("error creating users table: %v", err)
	}

	// Create tweets table
	if _, err := db.Exec(createTweetsTable); err != nil {
		return fmt.Errorf("error creating tweets table: %v", err)
	}

	// Create smart_users table
	if _, err := db.Exec(createSmartUsersTable); err != nil {
		return fmt.Errorf("error creating smart_users table: %v", err)
	}

	// Create smart_tweets table
	if _, err := db.Exec(createSmartTweetsTable); err != nil {
		return fmt.Errorf("error creating smart_tweets table: %v", err)
	}

	// Create text indexes for tweets table
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_tweets_text ON tweets USING gin(to_tsvector('english', text))"); err != nil {
		return fmt.Errorf("error creating text index for tweets table: %v", err)
	}

	// Create text indexes for smart_tweets table
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_smart_tweets_text ON smart_tweets USING gin(to_tsvector('english', text))"); err != nil {
		return fmt.Errorf("error creating text index for smart_tweets table: %v", err)
	}

	return nil
}

// func insertUsernames(db *sql.DB, usernames []string) error {
// 	// Insert usernames if they don't exist
// 	for _, username := range usernames {
// 		var exists bool
// 		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE Username = $1)", username).Scan(&exists)
// 		if err != nil {
// 			return fmt.Errorf("error checking username existence: %v", err)
// 		}

// 		if !exists {
// 			_, err = db.Exec("INSERT INTO users (Username) VALUES ($1)", username)
// 			if err != nil {
// 				return fmt.Errorf("error inserting username: %v", err)
// 			}
// 			log.Printf("Inserted username: %s", username)
// 		}
// 	}
// 	return nil
// }

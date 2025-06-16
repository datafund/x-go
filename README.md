# X-Go

A Go library and HTTP server for interacting with Twitter/X using the `github.com/imperatrona/twitter-scraper` package. The service also includes a PostgreSQL database for storing and searching tweets.

## Account Management

The library supports managing multiple Twitter accounts with cookie persistence. Here's how it works:

1. Create an `accounts.json` file in the XGO path (default: `$HOME/x-go`) with your Twitter accounts:
   ```json
   [
       {
           "username": "your_twitter_username",
           "password": "your_twitter_password"
       }
   ]
   ```

2. When the server starts:
   - It checks for existing cookies in the `cookies` directory
   - If cookies exist, it tries to use them for authentication
   - If cookies are invalid or don't exist, it logs in using the credentials from `accounts.json`
   - After successful login, it saves the cookies to `cookies/{username}.json`

## Environment Variables

- `XGO_PATH`: Path to the X-Go directory (default: `$HOME/x-go`)

## Database Configuration

Create a `config.yaml` file in the root directory with the following structure:

```yaml
usernames:
  - username1
  - username2
  # Add more usernames to track

postgres_url: "postgres://username:password@localhost:5432/dbname"
getmoni_api_key: "your_getmoni_api_key" # Required for GetMoni API integration
```

### Database Migration

Before running the server for the first time or after making changes to the database schema, run the migration command:

```bash
go run cmd/migrate/main.go
```

This will:
1. Create the necessary database tables if they don't exist
2. Insert usernames from config.yaml into the users table
3. Set up indexes and constraints

### GetMoni API Integration

The service integrates with the GetMoni API for additional functionality. To use this feature:

1. Obtain an API key from GetMoni
2. Add the `getmoni_api_key` to your `config.yaml`

## API Endpoints

### Public Endpoints (No Login Required)
- `GET /api/user/{username}/tweets` - Get user tweets
- `GET /api/user/{username}/profile` - Get user profile
- `GET /api/tweet/{id}` - Get tweet by ID
- `GET /api/search/tweets` - Search tweets in database
  - Query parameters:
    - `q` (required) - Search query
    - `sort_by` (optional) - Sort by "timestamp", "likes", or "views"
    - `limit` (optional) - Number of tweets to return (default: 50)

### Authenticated Endpoints (Login Required)
- `GET /api/search?q={query}` - Search tweets
- `POST /api/follow/{id}` - Follow user
- `POST /api/unfollow/{id}` - Unfollow user
- `POST /api/tweet` - Create tweet
- `POST /api/tweet/{id}/like` - Like tweet
- `POST /api/tweet/{id}/unlike` - Unlike tweet
- `POST /api/tweet/{id}/retweet` - Retweet

## Background Tasks

The service runs two background tasks:

1. Profile Updates: Updates user profiles every 10 seconds
2. Tweet Updates: Fetches 20 tweets per user every 6 hours

## MCP Server

The project implements a Multi-Agent Communication Protocol (MCP) server that provides programmatic access to Twitter functionality through standardized agent communication.

### MCP Server Features
- Manages multiple Twitter agents with session persistence
- Provides tool-based interaction with Twitter API
- Supports middleware for request handling
- Includes logging and recovery capabilities

### Environment Variables
- `XGO_PATH`: Path to the X-Go directory (default: `$HOME/x-go`) - Required for agent management and cookie storage

### Running as MCP Server

1. Ensure `XGO_PATH` environment variable is set
2. Configure your Twitter accounts in `accounts.json`
3. Run the MCP server:
   ```bash
   go run main.go
   ```

The server will start and handle MCP protocol communication through stdin/stdout.

## Building and Running Servers

This project supports two server modes: HTTP API server and MCP server.

### Building the Servers

1. Build the HTTP server:
   ```bash
   go build -o x-go-http cmd/httpserver/main.go
   ```

2. Build the MCP server:
   ```bash
   go build -o x-go-mcp main.go
   ```

3. Build the migration tool:
   ```bash
   go build -o x-go-migrate cmd/migrate/main.go
   ```

### Running HTTP Server

1. Copy `accounts.json.example` to `accounts.json` and add your Twitter accounts
2. Create `config.yaml` with your database configuration and usernames to track
3. Run database migrations:
   ```bash
   ./x-go-migrate
   ```
4. Set environment variables:
   ```bash
   export XGO_PATH=$HOME/x-go
   ```
5. Run the server:
   ```bash
   ./x-go-http
   ```

The HTTP server will start on port 8080.

### Running MCP Server

1. Ensure `accounts.json` is configured with your Twitter accounts
2. Set environment variables:
   ```bash
   export XGO_PATH=$HOME/x-go
   ```
3. Run the server:
   ```bash
   ./x-go-mcp
   ```

The MCP server will handle communication through stdin/stdout using the MCP protocol.

### Docker Support

You can also run the servers using Docker. Both servers use a Docker volume to persist data and configurations.

1. Build HTTP server image:
   ```bash
   docker build -f Dockerfile.http -t x-go-http .
   ```

2. Build MCP server image:
   ```bash
   docker build -f Dockerfile.mcp -t x-go-mcp .
   ```

3. Create a Docker volume for data persistence (optional):
   ```bash
   docker volume create x-go-data
   ```

4. Run database migrations:
   ```bash
   docker run --rm \
     -v x-go-data:/x-go \
     x-go-http ./x-go-migrate
   ```

5. Run HTTP server container:
   ```bash
   # Using a named volume
   docker run -p 8080:8080 \
     -v x-go-data:/x-go \
     x-go-http

   # OR using a local directory
   docker run -p 8080:8080 \
     -v $HOME/x-go:/x-go \
     x-go-http
   ```

6. Run MCP server container:
   ```bash
   # Using a named volume
   docker run -i \
     -v x-go-data:/x-go \
     x-go-mcp

   # OR using a local directory
   docker run -i \
     -v $HOME/x-go:/x-go \
     x-go-mcp
   ```

The volume at `/x-go` contains:
- `accounts.json`: Twitter account credentials
- `cookies/`: Directory storing authentication cookies
- `config.yaml`: Database configuration and usernames to track
- Other persistent data generated by the application

Note: 
- The `-v` flag mounts the volume to persist data between container restarts
- Using a named volume (`x-go-data`) is recommended for production
- Using a local directory mount is useful for development
- The same volume can be shared between HTTP and MCP servers if needed 
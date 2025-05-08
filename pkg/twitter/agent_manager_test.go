package twitter

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAgentManager(t *testing.T) {
	agents := []*Agent{
		newMockAgent(),
		newMockAgent(),
		newMockAgent(),
	}

	am := NewAgentManager(agents)
	assert.NotNil(t, am)
	assert.Equal(t, len(agents), am.GetAgentCount())
}

func TestGetNextAgent(t *testing.T) {
	agents := []*Agent{
		newMockAgent(),
		newMockAgent(),
		newMockAgent(),
	}

	am := NewAgentManager(agents)

	// Test round-robin behavior
	firstAgent := am.getNextAgent()
	secondAgent := am.getNextAgent()
	thirdAgent := am.getNextAgent()
	fourthAgent := am.getNextAgent() // Should wrap around to first agent

	assert.NotEqual(t, firstAgent, secondAgent)
	assert.NotEqual(t, secondAgent, thirdAgent)
	assert.Equal(t, firstAgent, fourthAgent) // Should be the same as first agent
}

func TestSetCookies(t *testing.T) {
	agents := []*Agent{
		newMockAgent(),
		newMockAgent(),
	}

	am := NewAgentManager(agents)

	cookies := []*http.Cookie{
		{Name: "test", Value: "value"},
	}

	// Test valid index
	err := am.SetCookies(0, cookies)
	assert.NoError(t, err)

	// Test invalid indices
	err = am.SetCookies(-1, cookies)
	assert.Equal(t, ErrInvalidAgentIndex, err)

	err = am.SetCookies(len(agents), cookies)
	assert.Equal(t, ErrInvalidAgentIndex, err)
}

func TestGetAgent(t *testing.T) {
	agents := []*Agent{
		newMockAgent(),
		newMockAgent(),
	}

	am := NewAgentManager(agents)

	// Test valid index
	agent, err := am.GetAgent(0)
	assert.NoError(t, err)
	assert.NotNil(t, agent)

	// Test invalid indices
	agent, err = am.GetAgent(-1)
	assert.Equal(t, ErrInvalidAgentIndex, err)
	assert.Nil(t, agent)

	agent, err = am.GetAgent(len(agents))
	assert.Equal(t, ErrInvalidAgentIndex, err)
	assert.Nil(t, agent)
}

func TestAgentManagerOperations(t *testing.T) {
	agents := []*Agent{
		newMockAgent(),
		newMockAgent(),
	}

	am := NewAgentManager(agents)
	ctx := context.Background()

	// Test all operations to ensure they work through the manager
	t.Run("GetUserTweets", func(t *testing.T) {
		result, err := am.GetUserTweets(ctx, "testuser", 10, false)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("GetProfile", func(t *testing.T) {
		result, err := am.GetProfile(ctx, "testuser")
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("GetTweet", func(t *testing.T) {
		result, err := am.GetTweet(ctx, "123")
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("SearchTweets", func(t *testing.T) {
		result, err := am.SearchTweets(ctx, "test", 10)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("CreateTweet", func(t *testing.T) {
		result, err := am.CreateTweet(ctx, "test tweet", "")
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("LikeTweet", func(t *testing.T) {
		err := am.LikeTweet(ctx, "123")
		assert.NoError(t, err)
	})

	t.Run("UnlikeTweet", func(t *testing.T) {
		err := am.UnlikeTweet(ctx, "123")
		assert.NoError(t, err)
	})

	t.Run("Retweet", func(t *testing.T) {
		err := am.Retweet(ctx, "123")
		assert.NoError(t, err)
	})

	t.Run("Follow", func(t *testing.T) {
		err := am.Follow(ctx, "123")
		assert.NoError(t, err)
	})

	t.Run("Unfollow", func(t *testing.T) {
		err := am.Unfollow(ctx, "123")
		assert.NoError(t, err)
	})
}

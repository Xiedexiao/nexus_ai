package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCacheHitStatsByModel(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Exec("DELETE FROM quota_data").Error)
	t.Cleanup(func() { DB.Exec("DELETE FROM quota_data") })

	now := time.Now().Unix()
	rows := []QuotaData{
		// User 1, 50% hit rate: read 100, write 0, uncached 100.
		{UserID: 1, ModelName: "gpt-x", Count: 2, CacheReadTokens: 100, CacheCreationTokens: 0, TokenUsed: 100, CreatedAt: now},
		// User 1, never touched the cache: denominator falls back to prompt tokens only.
		{UserID: 1, ModelName: "no-cache-model", Count: 1, CacheReadTokens: 0, CacheCreationTokens: 0, TokenUsed: 300, CreatedAt: now},
		// User 1, fully cached.
		{UserID: 1, ModelName: "cached-model", Count: 3, CacheReadTokens: 500, CacheCreationTokens: 100, TokenUsed: 50, CreatedAt: now - 3600},
		// User 1, outside the window: must not appear.
		{UserID: 1, ModelName: "too-old", Count: 9, CacheReadTokens: 999, CreatedAt: now - 48*3600},
		// User 2 uses the same model: sums must stay isolated per user.
		{UserID: 2, ModelName: "gpt-x", Count: 5, CacheReadTokens: 10, CacheCreationTokens: 0, TokenUsed: 90, CreatedAt: now},
	}
	for i := range rows {
		require.NoError(t, DB.Create(&rows[i]).Error)
	}

	stats, err := GetCacheHitStatsByModel(1, now-3600*6, now)
	require.NoError(t, err)
	require.Len(t, stats, 3)

	byModel := map[string]ModelCacheHitStat{}
	for _, st := range stats {
		byModel[st.ModelName] = st
	}

	gpt := byModel["gpt-x"]
	require.NotNil(t, &gpt)
	assert.Equal(t, int64(2), gpt.RequestCount)
	assert.InDelta(t, 0.5, gpt.CacheHitRate, 0.0001)

	nocache := byModel["no-cache-model"]
	assert.Equal(t, int64(0), nocache.CacheReadTokens)
	assert.Equal(t, 0.0, nocache.CacheHitRate, "no cache activity must yield zero, not NaN")

	cached := byModel["cached-model"]
	assert.InDelta(t, 500.0/650.0, cached.CacheHitRate, 0.0001)

	_, exists := byModel["too-old"]
	assert.False(t, exists, "rows outside the window must be excluded")

	// Ordering: highest request count first.
	assert.Equal(t, "cached-model", stats[0].ModelName)

	// Per-user scoping: user 2 only sees their own rows for the same model.
	stats2, err := GetCacheHitStatsByModel(2, now-3600*6, now)
	require.NoError(t, err)
	require.Len(t, stats2, 1)
	assert.Equal(t, "gpt-x", stats2[0].ModelName)
	assert.Equal(t, int64(5), stats2[0].RequestCount)
	assert.Equal(t, int64(10), stats2[0].CacheReadTokens)
	assert.InDelta(t, 0.1, stats2[0].CacheHitRate, 0.0001)

	// A user with no usage rows gets an empty result, not another user's data.
	empty, err := GetCacheHitStatsByModel(3, now-3600*6, now)
	require.NoError(t, err)
	require.Empty(t, empty)
}

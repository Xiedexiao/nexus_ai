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
		// 50% hit rate: read 100, write 0, uncached 100.
		{ModelName: "gpt-x", Count: 2, CacheReadTokens: 100, CacheCreationTokens: 0, TokenUsed: 100, CreatedAt: now},
		// Never touched the cache: denominator falls back to prompt tokens only.
		{ModelName: "no-cache-model", Count: 1, CacheReadTokens: 0, CacheCreationTokens: 0, TokenUsed: 300, CreatedAt: now},
		// Fully cached.
		{ModelName: "cached-model", Count: 3, CacheReadTokens: 500, CacheCreationTokens: 100, TokenUsed: 50, CreatedAt: now - 3600},
		// Outside the window: must not appear.
		{ModelName: "too-old", Count: 9, CacheReadTokens: 999, CreatedAt: now - 48*3600},
	}
	for i := range rows {
		require.NoError(t, DB.Create(&rows[i]).Error)
	}

	stats, err := GetCacheHitStatsByModel(now-3600*6, now)
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
}

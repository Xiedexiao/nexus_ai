package model

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelayTimeoutOptionsUpdateRuntimeValues(t *testing.T) {
	// updateOptionMap writes through common.OptionMap, which is only
	// initialized by InitOptionMap at startup; provide one for the test.
	originalMap := common.OptionMap
	common.OptionMap = map[string]string{}
	t.Cleanup(func() { common.OptionMap = originalMap })

	originalInts := map[string]int{
		"RelayTimeout":               common.RelayTimeout,
		"RelayIdleConnTimeout":       common.RelayIdleConnTimeout,
		"RelayResponseHeaderTimeout": common.RelayResponseHeaderTimeout,
		"RelayMaxIdleConns":          common.RelayMaxIdleConns,
		"RelayMaxIdleConnsPerHost":   common.RelayMaxIdleConnsPerHost,
		"StreamingTimeout":           constant.StreamingTimeout,
		"CriticalRateLimitNum":       common.CriticalRateLimitNum,
		"TaskTimeoutMinutes":         constant.TaskTimeoutMinutes,
		"TaskPollMaxFailures":        constant.TaskPollMaxFailures,
		"VerificationValidMinutes":   common.VerificationValidMinutes,
	}
	originalWebEnable := common.GlobalWebRateLimitEnable
	originalCriticalDuration := common.CriticalRateLimitDuration
	restore := func() {
		common.RelayTimeout = originalInts["RelayTimeout"]
		common.RelayIdleConnTimeout = originalInts["RelayIdleConnTimeout"]
		common.RelayResponseHeaderTimeout = originalInts["RelayResponseHeaderTimeout"]
		common.RelayMaxIdleConns = originalInts["RelayMaxIdleConns"]
		common.RelayMaxIdleConnsPerHost = originalInts["RelayMaxIdleConnsPerHost"]
		constant.StreamingTimeout = originalInts["StreamingTimeout"]
		common.CriticalRateLimitNum = originalInts["CriticalRateLimitNum"]
		constant.TaskTimeoutMinutes = originalInts["TaskTimeoutMinutes"]
		constant.TaskPollMaxFailures = originalInts["TaskPollMaxFailures"]
		common.VerificationValidMinutes = originalInts["VerificationValidMinutes"]
		common.GlobalWebRateLimitEnable = originalWebEnable
		common.CriticalRateLimitDuration = originalCriticalDuration
	}
	t.Cleanup(restore)

	t.Run("a saved option becomes the live value", func(t *testing.T) {
		require.NoError(t, updateOptionMap("RelayTimeout", "321"))
		assert.Equal(t, 321, common.RelayTimeout)

		require.NoError(t, updateOptionMap("StreamingTimeout", "456"))
		assert.Equal(t, 456, constant.StreamingTimeout)

		require.NoError(t, updateOptionMap("RelayMaxIdleConnsPerHost", "77"))
		assert.Equal(t, 77, common.RelayMaxIdleConnsPerHost)
	})

	t.Run("validation rejects out-of-range input before it is stored", func(t *testing.T) {
		for key, max := range relayTimeoutOptionBounds {
			require.Error(t, validateOptionValue(key, "-1"), key)
			require.Error(t, validateOptionValue(key, "not-a-number"), key)
			require.Error(t, validateOptionValue(key, "9223372036854775808"), key)
			require.NoError(t, validateOptionValue(key, strconv.Itoa(max)), key)
			require.NoError(t, validateOptionValue(key, "0"), key)
		}
	})

	t.Run("out-of-range values from legacy rows are clamped instead of overflowing", func(t *testing.T) {
		require.NoError(t, updateOptionMap("RelayTimeout", "-5"))
		assert.Equal(t, 0, common.RelayTimeout)

		// A large but int64-representable value clamps to the bound; values
		// beyond int64 or non-numeric text are ignored (covered below).
		require.NoError(t, updateOptionMap("RelayResponseHeaderTimeout", "9999999999"))
		assert.Equal(t, maxRelayTimeoutSeconds, common.RelayResponseHeaderTimeout)

		require.NoError(t, updateOptionMap("RelayMaxIdleConns", "-1"))
		assert.Equal(t, 0, common.RelayMaxIdleConns)
	})

	t.Run("non-numeric input leaves the current value untouched", func(t *testing.T) {
		before := common.RelayIdleConnTimeout
		require.NoError(t, updateOptionMap("RelayIdleConnTimeout", "not-a-number"))
		assert.Equal(t, before, common.RelayIdleConnTimeout)
	})

	t.Run("rate limit task and verification options round-trip", func(t *testing.T) {
		require.NoError(t, updateOptionMap("CriticalRateLimitNum", "30"))
		assert.Equal(t, 30, common.CriticalRateLimitNum)

		require.NoError(t, updateOptionMap("CriticalRateLimitDuration", "600"))
		assert.Equal(t, int64(600), common.CriticalRateLimitDuration)

		require.NoError(t, updateOptionMap("GlobalWebRateLimitEnable", "false"))
		assert.False(t, common.GlobalWebRateLimitEnable)

		require.NoError(t, updateOptionMap("TaskTimeoutMinutes", "0"))
		assert.Equal(t, 0, constant.TaskTimeoutMinutes)

		require.NoError(t, updateOptionMap("TaskPollMaxFailures", "5"))
		assert.Equal(t, 5, constant.TaskPollMaxFailures)

		require.NoError(t, updateOptionMap("VerificationValidMinutes", "15"))
		assert.Equal(t, 15, common.VerificationValidMinutes)

		for key, bounds := range adminTunableIntBounds {
			require.Error(t, validateOptionValue(key, strconv.Itoa(bounds[0]-1)), key)
			require.Error(t, validateOptionValue(key, strconv.Itoa(bounds[1]+1)), key)
			require.NoError(t, validateOptionValue(key, strconv.Itoa(bounds[0])), key)
			require.NoError(t, validateOptionValue(key, strconv.Itoa(bounds[1])), key)
		}
		for key := range adminTunableBoolOptions {
			require.Error(t, validateOptionValue(key, "yes"), key)
			require.NoError(t, validateOptionValue(key, "true"), key)
			require.NoError(t, validateOptionValue(key, "false"), key)
		}
	})

	t.Run("startup load applies the saved row over the environment seed", func(t *testing.T) {
		// loadOptionsFromDatabase routes every row through updateOptionMap, so a
		// persisted console value must win over the env-seeded default.
		common.RelayTimeout = 0
		require.NoError(t, updateOptionMap("RelayTimeout", "90"))
		assert.Equal(t, 90, common.RelayTimeout)
	})
}

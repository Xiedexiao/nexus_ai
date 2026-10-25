package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerificationCodeLifecycle(t *testing.T) {
	key := "verify-test@example.com"

	RegisterVerificationCodeWithKey(key, "abc123", EmailVerificationPurpose)

	t.Run("accepts the correct code", func(t *testing.T) {
		assert.True(t, VerifyCodeWithKey(key, "abc123", EmailVerificationPurpose))
	})

	t.Run("rejects a wrong code without consuming it", func(t *testing.T) {
		assert.False(t, VerifyCodeWithKey(key, "wrong9", EmailVerificationPurpose))
		// The real code still works afterwards, so one typo does not lock the
		// user out; request-level rate limiting is the brute-force control.
		assert.True(t, VerifyCodeWithKey(key, "abc123", EmailVerificationPurpose))
	})

	t.Run("rejects a code issued for another purpose", func(t *testing.T) {
		assert.False(t, VerifyCodeWithKey(key, "abc123", PasswordResetPurpose))
	})

	t.Run("delete removes the code", func(t *testing.T) {
		DeleteKey(key, EmailVerificationPurpose)
		assert.False(t, VerifyCodeWithKey(key, "abc123", EmailVerificationPurpose))
	})
}

func TestVerificationCodeComparisonIsLengthIndependent(t *testing.T) {
	key := "verify-length@example.com"
	RegisterVerificationCodeWithKey(key, "abc123", EmailVerificationPurpose)

	// A longer guess must fail cleanly; the constant-time compare must not
	// panic or match on any length mismatch, including the empty string.
	require.False(t, VerifyCodeWithKey(key, "", EmailVerificationPurpose))
	require.False(t, VerifyCodeWithKey(key, "abc1234", EmailVerificationPurpose))
	require.False(t, VerifyCodeWithKey(key, "abc12", EmailVerificationPurpose))
	assert.True(t, VerifyCodeWithKey(key, "abc123", EmailVerificationPurpose))
}

func TestRegisterVerificationCodeDoesNotEvictLiveCodes(t *testing.T) {
	// The size cap only triggers expired-entry cleanup. Filling the map with
	// fresh codes must not evict an earlier still-valid one; that behaviour is
	// load bearing for concurrent sign-ups, so this test pins it.
	first := "first@example.com"
	RegisterVerificationCodeWithKey(first, "111111", EmailVerificationPurpose)
	for i := range 20 {
		RegisterVerificationCodeWithKey(liveCodeKey(i), "222222", EmailVerificationPurpose)
	}
	assert.True(t, VerifyCodeWithKey(first, "111111", EmailVerificationPurpose))
	for i := range 20 {
		DeleteKey(liveCodeKey(i), EmailVerificationPurpose)
	}
	DeleteKey(first, EmailVerificationPurpose)
}

func liveCodeKey(i int) string {
	return "bulk-" + string(rune('a'+i)) + "@example.com"
}

package model

import (
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertRegisteredUser(t *testing.T, username, ip string) {
	t.Helper()
	require.NoError(t, DB.Create(&User{
		Username:   username,
		Password:   "irrelevant-hash",
		AffCode:    "rip" + username,
		RegisterIp: ip,
		Role:       common.RoleCommonUser,
		Status:     common.UserStatusEnabled,
	}).Error)
}

func TestRegistrationIpLimitGate(t *testing.T) {
	truncateTables(t)

	originalLimit := common.RegistrationIpLimit
	t.Cleanup(func() { common.RegistrationIpLimit = originalLimit })

	t.Run("a limit of zero disables the cap", func(t *testing.T) {
		clearUsersByIp(t, "10.0.0.1")
		common.RegistrationIpLimit = 0
		insertRegisteredUser(t, "cap-off-1", "10.0.0.1")
		insertRegisteredUser(t, "cap-off-2", "10.0.0.1")
		require.NoError(t, EnsureRegistrationIpAllowed("10.0.0.1"))
	})

	t.Run("blocks the address once the limit is reached", func(t *testing.T) {
		clearUsersByIp(t, "10.0.0.2")
		common.RegistrationIpLimit = 2
		require.NoError(t, EnsureRegistrationIpAllowed("10.0.0.2"))
		insertRegisteredUser(t, "cap-1", "10.0.0.2")
		require.NoError(t, EnsureRegistrationIpAllowed("10.0.0.2"))
		insertRegisteredUser(t, "cap-2", "10.0.0.2")
		require.ErrorIs(t, EnsureRegistrationIpAllowed("10.0.0.2"), ErrRegistrationIpLimitReached)
	})

	t.Run("counts only that address", func(t *testing.T) {
		clearUsersByIp(t, "10.0.0.3")
		clearUsersByIp(t, "10.0.0.4")
		common.RegistrationIpLimit = 1
		insertRegisteredUser(t, "other-1", "10.0.0.4")
		require.NoError(t, EnsureRegistrationIpAllowed("10.0.0.3"))
		insertRegisteredUser(t, "mine-1", "10.0.0.3")
		require.ErrorIs(t, EnsureRegistrationIpAllowed("10.0.0.3"), ErrRegistrationIpLimitReached)
		// The other address is capped independently.
		require.ErrorIs(t, EnsureRegistrationIpAllowed("10.0.0.4"), ErrRegistrationIpLimitReached)
	})

	t.Run("accounts without a recorded ip never count", func(t *testing.T) {
		clearUsersByIp(t, "")
		common.RegistrationIpLimit = 1
		require.NoError(t, DB.Create(&User{
			Username: "legacy-no-ip",
			Password: "irrelevant-hash",
			AffCode:  "riplegacy",
			Role:     common.RoleRootUser,
			Status:   common.UserStatusEnabled,
		}).Error)
		require.NoError(t, EnsureRegistrationIpAllowed("10.0.0.5"))
	})

	t.Run("an empty caller address is allowed", func(t *testing.T) {
		common.RegistrationIpLimit = 1
		require.NoError(t, EnsureRegistrationIpAllowed(""))
	})

	t.Run("registration ip is not serialized to clients", func(t *testing.T) {
		clearUsersByIp(t, "10.0.0.6")
		insertRegisteredUser(t, "json-check", "10.0.0.6")
		var user User
		require.NoError(t, DB.Where("username = ?", "json-check").First(&user).Error)
		assert.Contains(t, user.toJSON(t), `"username":"json-check"`)
		assert.NotContains(t, user.toJSON(t), "register_ip")
		assert.NotContains(t, user.toJSON(t), "10.0.0.6")
	})
}

func (u *User) toJSON(t *testing.T) string {
	t.Helper()
	data, err := common.Marshal(u)
	require.NoError(t, err)
	return string(data)
}

func clearUsersByIp(t *testing.T, ip string) {
	t.Helper()
	if ip == "" {
		require.NoError(t, DB.Exec("DELETE FROM users WHERE register_ip IS NULL OR register_ip = ''").Error)
		return
	}
	require.NoError(t, DB.Exec("DELETE FROM users WHERE register_ip = ?", ip).Error)
}

func insertWindowUser(t *testing.T, username, ip string, createdAt int64) {
	t.Helper()
	require.NoError(t, DB.Create(&User{
		Username:   username,
		Password:   "irrelevant-hash",
		AffCode:    "win" + username,
		RegisterIp: ip,
		CreatedAt:  createdAt,
		Role:       common.RoleCommonUser,
		Status:     common.UserStatusEnabled,
	}).Error)
}

func TestRegistrationWindowGate(t *testing.T) {
	truncateTables(t)

	originalLimit := common.RegistrationWindowLimit
	originalMinutes := common.RegistrationWindowMinutes
	t.Cleanup(func() {
		common.RegistrationWindowLimit = originalLimit
		common.RegistrationWindowMinutes = originalMinutes
	})

	t.Run("a limit of zero disables the cap", func(t *testing.T) {
		common.RegistrationWindowLimit = 0
		common.RegistrationWindowMinutes = 60
		insertWindowUser(t, "win-off-1", "10.1.0.1", common.GetTimestamp())
		insertWindowUser(t, "win-off-2", "10.1.0.2", common.GetTimestamp())
		require.NoError(t, EnsureRegistrationWindowAllowed())
	})

	t.Run("blocks once the window admits its quota", func(t *testing.T) {
		require.NoError(t, DB.Exec("DELETE FROM users").Error)
		common.RegistrationWindowLimit = 2
		common.RegistrationWindowMinutes = 60
		require.NoError(t, EnsureRegistrationWindowAllowed())
		insertWindowUser(t, "win-1", "10.1.1.1", common.GetTimestamp())
		insertWindowUser(t, "win-2", "10.1.1.2", common.GetTimestamp())
		require.ErrorIs(t, EnsureRegistrationWindowAllowed(), ErrRegistrationWindowReached)
	})

	t.Run("accounts outside the window no longer count", func(t *testing.T) {
		require.NoError(t, DB.Exec("DELETE FROM users").Error)
		common.RegistrationWindowLimit = 2
		common.RegistrationWindowMinutes = 60
		old := common.GetTimestamp() - 61*60
		insertWindowUser(t, "win-old-1", "10.1.2.1", old)
		insertWindowUser(t, "win-old-2", "10.1.2.2", old)
		require.NoError(t, EnsureRegistrationWindowAllowed())
		// New sign-ups inside the window still count against the cap.
		insertWindowUser(t, "win-new-1", "10.1.2.3", common.GetTimestamp())
		require.NoError(t, EnsureRegistrationWindowAllowed())
		insertWindowUser(t, "win-new-2", "10.1.2.4", common.GetTimestamp())
		require.ErrorIs(t, EnsureRegistrationWindowAllowed(), ErrRegistrationWindowReached)
	})

	t.Run("accounts without a recorded ip never consume the window", func(t *testing.T) {
		require.NoError(t, DB.Exec("DELETE FROM users").Error)
		common.RegistrationWindowLimit = 1
		common.RegistrationWindowMinutes = 60
		require.NoError(t, DB.Create(&User{
			Username:  "win-admin-made",
			Password:  "irrelevant-hash",
			AffCode:   "winadmin",
			CreatedAt: common.GetTimestamp(),
			Role:      common.RoleRootUser,
			Status:    common.UserStatusEnabled,
		}).Error)
		require.NoError(t, EnsureRegistrationWindowAllowed())
	})
}

func TestRegistrationIpLimitOption(t *testing.T) {
	originalMap := common.OptionMap
	common.OptionMap = map[string]string{}
	t.Cleanup(func() { common.OptionMap = originalMap })

	originalLimit := common.RegistrationIpLimit
	originalWindowLimit := common.RegistrationWindowLimit
	originalWindowMinutes := common.RegistrationWindowMinutes
	t.Cleanup(func() {
		common.RegistrationIpLimit = originalLimit
		common.RegistrationWindowLimit = originalWindowLimit
		common.RegistrationWindowMinutes = originalWindowMinutes
	})

	require.NoError(t, updateOptionMap("RegistrationIpLimit", "3"))
	assert.Equal(t, 3, common.RegistrationIpLimit)

	require.Error(t, validateOptionValue("RegistrationIpLimit", "-1"))
	require.Error(t, validateOptionValue("RegistrationIpLimit", "not-a-number"))
	require.Error(t, validateOptionValue("RegistrationIpLimit", strconv.Itoa(maxRegistrationIpLimit+1)))
	require.NoError(t, validateOptionValue("RegistrationIpLimit", "0"))
	require.NoError(t, validateOptionValue("RegistrationIpLimit", strconv.Itoa(maxRegistrationIpLimit)))

}

func TestStaleAccountHousekeeping(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Exec("DELETE FROM users").Error)

	// Eligible: common user, never logged in, no balance, old enough.
	old := time.Now().Unix() - 100*24*60*60
	require.NoError(t, DB.Create(&User{
		Username: "stale-1", Password: "x", AffCode: "st0001",
		Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
		CreatedAt: old,
	}).Error)
	// Protected: admin role.
	require.NoError(t, DB.Create(&User{
		Username: "stale-admin", Password: "x", AffCode: "st0002",
		Role: common.RoleRootUser, Status: common.UserStatusEnabled,
		CreatedAt: old,
	}).Error)
	// Protected: logged in before.
	require.NoError(t, DB.Create(&User{
		Username: "stale-active", Password: "x", AffCode: "st0003",
		Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
		CreatedAt: old, LastLoginAt: time.Now().Unix(),
	}).Error)
	// Protected: has balance.
	require.NoError(t, DB.Create(&User{
		Username: "stale-funded", Password: "x", AffCode: "st0004",
		Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
		CreatedAt: old, Quota: 500000,
	}).Error)
	// Protected: too recent.
	require.NoError(t, DB.Create(&User{
		Username: "stale-fresh", Password: "x", AffCode: "st0005",
		Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
		CreatedAt: time.Now().Unix(),
	}).Error)

	cutoff := time.Now().Unix() - 90*24*60*60
	count, err := CountStaleAccounts(StaleAccountCriteria{CreatedBefore: cutoff})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	deleted, err := DeleteStaleAccounts(StaleAccountCriteria{CreatedBefore: cutoff, Limit: 100})
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	var remaining int64
	require.NoError(t, DB.Model(&User{}).Count(&remaining).Error)
	assert.Equal(t, int64(4), remaining, "only the abandoned common user may be removed")
}

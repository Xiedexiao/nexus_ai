package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setBootstrapEnv(t *testing.T, username, password string) {
	t.Helper()
	t.Setenv("INITIAL_ROOT_USERNAME", username)
	t.Setenv("INITIAL_ROOT_PASSWORD", password)
	t.Setenv("INITIAL_ROOT_EMAIL", "")
}

func setBootstrapEnvWithEmail(t *testing.T, username, password, email string) {
	t.Helper()
	t.Setenv("INITIAL_ROOT_USERNAME", username)
	t.Setenv("INITIAL_ROOT_PASSWORD", password)
	t.Setenv("INITIAL_ROOT_EMAIL", email)
}

// clearUsers empties the table immediately so each case starts from a known
// state; the shared truncateTables helper only deletes during cleanup.
func clearUsers(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Exec("DELETE FROM users").Error)
}

func countRootUsers(t *testing.T) int64 {
	t.Helper()
	var count int64
	require.NoError(t, DB.Model(&User{}).Where("role = ?", common.RoleRootUser).Count(&count).Error)
	return count
}

func TestBootstrapRootAccountFromEnv(t *testing.T) {
	truncateTables(t)

	t.Run("creates the administrator when no root user exists", func(t *testing.T) {
		clearUsers(t)
		setBootstrapEnv(t, "root-admin", "bootstrap-password-1")

		require.NoError(t, BootstrapRootAccountFromEnv())

		var user User
		require.NoError(t, DB.Where("username = ?", "root-admin").First(&user).Error)
		assert.Equal(t, common.RoleRootUser, user.Role)
		assert.Equal(t, common.UserStatusEnabled, user.Status)
		assert.True(t, common.ValidatePasswordAndHash("bootstrap-password-1", user.Password))
		assert.NotEqual(t, "bootstrap-password-1", user.Password)
	})

	t.Run("is idempotent across restarts", func(t *testing.T) {
		clearUsers(t)
		setBootstrapEnv(t, "root-admin", "bootstrap-password-1")

		require.NoError(t, BootstrapRootAccountFromEnv())
		require.NoError(t, BootstrapRootAccountFromEnv())

		assert.Equal(t, int64(1), countRootUsers(t))
	})

	t.Run("never replaces an existing root user", func(t *testing.T) {
		clearUsers(t)
		setBootstrapEnv(t, "root-admin", "bootstrap-password-1")
		require.NoError(t, BootstrapRootAccountFromEnv())

		setBootstrapEnv(t, "other-admin", "another-password-2")
		require.NoError(t, BootstrapRootAccountFromEnv())

		assert.Equal(t, int64(1), countRootUsers(t))
		var user User
		require.NoError(t, DB.Where("username = ?", "root-admin").First(&user).Error)
		assert.True(t, common.ValidatePasswordAndHash("bootstrap-password-1", user.Password))
	})

	t.Run("does nothing when the variables are unset", func(t *testing.T) {
		clearUsers(t)
		setBootstrapEnv(t, "", "")

		require.NoError(t, BootstrapRootAccountFromEnv())

		assert.Equal(t, int64(0), countRootUsers(t))
	})

	t.Run("rejects a partial configuration", func(t *testing.T) {
		clearUsers(t)
		setBootstrapEnv(t, "root-admin", "")

		require.Error(t, BootstrapRootAccountFromEnv())
		assert.Equal(t, int64(0), countRootUsers(t))
	})

	t.Run("rejects a username longer than the wizard allows", func(t *testing.T) {
		clearUsers(t)
		setBootstrapEnv(t, "thirteenchars", "bootstrap-password-1")

		require.Error(t, BootstrapRootAccountFromEnv())
		assert.Equal(t, int64(0), countRootUsers(t))
	})

	t.Run("rejects a password that violates the account policy", func(t *testing.T) {
		clearUsers(t)
		setBootstrapEnv(t, "root-admin", "short")

		require.Error(t, BootstrapRootAccountFromEnv())
		assert.Equal(t, int64(0), countRootUsers(t))
	})

	t.Run("stores the email so the account can sign in with it", func(t *testing.T) {
		clearUsers(t)
		setBootstrapEnvWithEmail(t, "root-admin", "bootstrap-password-1", " Admin@Example.COM ")

		require.NoError(t, BootstrapRootAccountFromEnv())

		var user User
		require.NoError(t, DB.Where("username = ?", "root-admin").First(&user).Error)
		assert.Equal(t, "admin@example.com", user.Email)

		// Login looks up username OR email, so the email must authenticate.
		byEmail := User{Username: "admin@example.com", Password: "bootstrap-password-1"}
		require.NoError(t, byEmail.ValidateAndFill())
		assert.Equal(t, user.Id, byEmail.Id)
	})

	t.Run("leaves the email empty when it is not configured", func(t *testing.T) {
		clearUsers(t)
		setBootstrapEnv(t, "root-admin", "bootstrap-password-1")

		require.NoError(t, BootstrapRootAccountFromEnv())

		var user User
		require.NoError(t, DB.Where("username = ?", "root-admin").First(&user).Error)
		assert.Empty(t, user.Email)
	})

	t.Run("rejects a malformed email", func(t *testing.T) {
		clearUsers(t)
		setBootstrapEnvWithEmail(t, "root-admin", "bootstrap-password-1", "not-an-email")

		require.Error(t, BootstrapRootAccountFromEnv())
		assert.Equal(t, int64(0), countRootUsers(t))
	})

	t.Run("rejects an email already used by another account", func(t *testing.T) {
		clearUsers(t)
		require.NoError(t, DB.Create(&User{
			Username: "taken-user",
			Password: "irrelevant-hash",
			Email:    "taken@example.com",
			AffCode:  "taken001",
			Role:     common.RoleCommonUser,
			Status:   common.UserStatusEnabled,
		}).Error)
		setBootstrapEnvWithEmail(t, "root-admin", "bootstrap-password-1", "TAKEN@example.com")

		require.Error(t, BootstrapRootAccountFromEnv())
		assert.Equal(t, int64(0), countRootUsers(t))
	})

	t.Run("attaches the email to an existing administrator that has none", func(t *testing.T) {
		clearUsers(t)
		// An administrator created before INITIAL_ROOT_EMAIL existed.
		hashed, err := common.HashAccountPassword("existing-password-1")
		require.NoError(t, err)
		require.NoError(t, DB.Create(&User{
			Username: "root-admin",
			Password: hashed,
			AffCode:  "root0001",
			Role:     common.RoleRootUser,
			Status:   common.UserStatusEnabled,
			Quota:    100000000,
		}).Error)

		setBootstrapEnvWithEmail(t, "ignored", "ignored-password-2", "admin@example.com")
		require.NoError(t, BootstrapRootAccountFromEnv())

		var user User
		require.NoError(t, DB.Where("username = ?", "root-admin").First(&user).Error)
		assert.Equal(t, "admin@example.com", user.Email)
		// The existing credentials must survive untouched.
		assert.True(t, common.ValidatePasswordAndHash("existing-password-1", user.Password))
		assert.Equal(t, int64(1), countRootUsers(t))

		byEmail := User{Username: "admin@example.com", Password: "existing-password-1"}
		require.NoError(t, byEmail.ValidateAndFill())
		assert.Equal(t, user.Id, byEmail.Id)
	})

	t.Run("never overwrites an administrator email that is already set", func(t *testing.T) {
		clearUsers(t)
		hashed, err := common.HashAccountPassword("existing-password-1")
		require.NoError(t, err)
		require.NoError(t, DB.Create(&User{
			Username: "root-admin",
			Password: hashed,
			Email:    "original@example.com",
			AffCode:  "root0002",
			Role:     common.RoleRootUser,
			Status:   common.UserStatusEnabled,
		}).Error)

		setBootstrapEnvWithEmail(t, "ignored", "ignored-password-2", "replacement@example.com")
		require.NoError(t, BootstrapRootAccountFromEnv())

		var user User
		require.NoError(t, DB.Where("username = ?", "root-admin").First(&user).Error)
		assert.Equal(t, "original@example.com", user.Email)
	})

	t.Run("does not attach an email owned by another account", func(t *testing.T) {
		clearUsers(t)
		hashed, err := common.HashAccountPassword("existing-password-1")
		require.NoError(t, err)
		require.NoError(t, DB.Create(&User{
			Username: "root-admin",
			Password: hashed,
			AffCode:  "root0003",
			Role:     common.RoleRootUser,
			Status:   common.UserStatusEnabled,
		}).Error)
		require.NoError(t, DB.Create(&User{
			Username: "some-user",
			Password: hashed,
			Email:    "taken@example.com",
			AffCode:  "some0001",
			Role:     common.RoleCommonUser,
			Status:   common.UserStatusEnabled,
		}).Error)

		setBootstrapEnvWithEmail(t, "ignored", "ignored-password-2", "taken@example.com")
		require.Error(t, BootstrapRootAccountFromEnv())

		var root User
		require.NoError(t, DB.Where("username = ?", "root-admin").First(&root).Error)
		assert.Empty(t, root.Email)
	})
}

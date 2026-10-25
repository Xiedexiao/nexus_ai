package model

import (
	"os"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestRegisterIpMigrationRealDatabases exercises the users.register_ip column
// on real MySQL and PostgreSQL instances (set TEST_MYSQL_DSN / TEST_POSTGRES_DSN
// to enable a dialect; unset dialects are skipped). It covers the fresh-database
// path, the upgrade path from a pre-column table, idempotent re-migration, and
// the exact COUNT query the per-IP registration gate runs. Each DSN must point
// at a throwaway database: the test drops and recreates the users table.
func TestRegisterIpMigrationRealDatabases(t *testing.T) {
	dialects := []struct {
		name string
		dsn  string
		open func(string) gorm.Dialector
	}{
		{"sqlite", t.TempDir() + "/reg_ip.db", func(dsn string) gorm.Dialector { return sqlite.Open(dsn) }},
		{"mysql", os.Getenv("TEST_MYSQL_DSN"), mysql.Open},
		{"postgres", os.Getenv("TEST_POSTGRES_DSN"), postgres.Open},
	}

	for _, d := range dialects {
		t.Run(d.name, func(t *testing.T) {
			if d.dsn == "" {
				t.Skipf("%s DSN not configured", d.name)
			}
			db, err := gorm.Open(d.open(d.dsn), &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { _ = sqlDB.Close() })

			require.NoError(t, db.Migrator().DropTable("users"))

			// Fresh database: the column and its index arrive with the table.
			require.NoError(t, db.AutoMigrate(&User{}))
			require.True(t, db.Migrator().HasColumn("users", "register_ip"), "fresh migration must add register_ip")
			require.True(t, db.Migrator().HasIndex("users", "idx_users_register_ip"), "fresh migration must index register_ip")
			require.True(t, db.Migrator().HasIndex("users", "idx_users_created_at"), "fresh migration must index created_at for the window count")

			// Upgrade path: simulate a table created before the column existed.
			// Raw statements, not the migrator helper: glebarez/sqlite's
			// DropColumn panics on its recreate-table path, and SQLite refuses
			// to drop a column that an index still references, so the index
			// goes first (MySQL and PostgreSQL drop it with the column).
			if db.Dialector.Name() == "mysql" {
				require.NoError(t, db.Exec("DROP INDEX idx_users_register_ip ON users").Error)
			} else {
				require.NoError(t, db.Exec("DROP INDEX IF EXISTS idx_users_register_ip").Error)
			}
			require.NoError(t, db.Exec("ALTER TABLE users DROP COLUMN register_ip").Error)
			require.False(t, db.Migrator().HasColumn("users", "register_ip"))
			require.NoError(t, db.AutoMigrate(&User{}))
			require.True(t, db.Migrator().HasColumn("users", "register_ip"), "upgrade must add register_ip to an existing table")
			require.True(t, db.Migrator().HasIndex("users", "idx_users_register_ip"), "upgrade must recreate the register_ip index")
			require.True(t, db.Migrator().HasIndex("users", "idx_users_created_at"), "upgrade must keep the created_at index")

			// Idempotency: migrating again changes nothing and errors nowhere.
			require.NoError(t, db.AutoMigrate(&User{}))
			require.True(t, db.Migrator().HasColumn("users", "register_ip"))

			// The gate's COUNT query behaves identically on every dialect.
			require.NoError(t, db.Create(&[]User{
				{Username: "rip-a1", Password: "x", AffCode: "ra1", RegisterIp: "203.0.113.9"},
				{Username: "rip-a2", Password: "x", AffCode: "ra2", RegisterIp: "203.0.113.9"},
				{Username: "rip-b1", Password: "x", AffCode: "rb1", RegisterIp: "198.51.100.7"},
			}).Error)

			var sameIp, otherIp, empty int64
			require.NoError(t, db.Table("users").Where("register_ip = ?", "203.0.113.9").Count(&sameIp).Error)
			require.NoError(t, db.Table("users").Where("register_ip = ?", "198.51.100.7").Count(&otherIp).Error)
			require.NoError(t, db.Table("users").Where("register_ip = ?", "").Count(&empty).Error)
			assert.Equal(t, int64(2), sameIp)
			assert.Equal(t, int64(1), otherIp)
			assert.Equal(t, int64(0), empty)

			// The window gate's query: self-service rows inside the cutoff.
			now := time.Now().Unix()
			require.NoError(t, db.Exec("UPDATE users SET created_at = ?", now-7200).Error)
			require.NoError(t, db.Exec("UPDATE users SET username = 'rip-a2', created_at = ? WHERE username = 'rip-a2'", now).Error)
			require.NoError(t, db.Exec("UPDATE users SET username = 'rip-b1', created_at = ? WHERE username = 'rip-b1'", now).Error)
			require.NoError(t, db.Create(&User{Username: "rip-old-noip", Password: "x", AffCode: "rn", CreatedAt: now}).Error)
			var inWindow int64
			require.NoError(t, db.Table("users").Where("register_ip IS NOT NULL AND register_ip <> '' AND created_at >= ?", now-60).Count(&inWindow).Error)
			assert.Equal(t, int64(2), inWindow)
		})
	}
}

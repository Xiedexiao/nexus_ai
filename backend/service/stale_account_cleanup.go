package service

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// Stale account housekeeping: sign-ups that never came back. A self-registered
// account counts as abandoned when it has never logged in after creation, has
// neither balance nor consumed quota, and predates the configured age. Deleted
// in small batches so a large backlog cannot lock the users table in one go.

const (
	staleAccountCleanupInterval = 24 * time.Hour
	staleAccountBatchSize       = 500
	StaleAccountMinAgeDaysFloor = 7
	staleAccountDefaultAgeDays  = 90
)

// StaleAccountCleanupAgeDays is admin-configurable via the options table
// (StaleAccountCleanupAgeDays). Values below the floor disable the sweep.
var StaleAccountCleanupAgeDays = 0

// StartStaleAccountCleanup runs the daily sweep on the master node only.
func StartStaleAccountCleanup() {
	if !common.IsMasterNode {
		return
	}
	go func() {
		ticker := time.NewTicker(staleAccountCleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			RunStaleAccountCleanup()
		}
	}()
}

// RunStaleAccountCleanup performs one batched deletion pass and logs the
// outcome. Safe to call manually from the admin API.
func RunStaleAccountCleanup() int64 {
	if StaleAccountCleanupAgeDays < StaleAccountMinAgeDaysFloor {
		return 0
	}
	cutoff := time.Now().Unix() - int64(StaleAccountCleanupAgeDays)*24*60*60
	criteria := model.StaleAccountCriteria{CreatedBefore: cutoff, Limit: staleAccountBatchSize}
	deleted, err := model.DeleteStaleAccounts(criteria)
	if err != nil {
		common.SysError(common.LogText("stale account cleanup failed: %s", err.Error()))
		return 0
	}
	if deleted > 0 {
		common.SysLog(common.LogText("stale account cleanup removed %d abandoned accounts (created before %d, never logged in, zero balance)", deleted, cutoff))
	}
	return deleted
}

//go:build synctest

package mcpscripttest

import (
	"testing/synctest"
)

const hasSynctest = true

// SynctestSupported returns true when synctest is available
func SynctestSupported() bool {
	return true
}

// RunWithSynctest executes a function with synctest for deterministic timing
func RunWithSynctest(f func()) {
	synctest.Run(f)
}
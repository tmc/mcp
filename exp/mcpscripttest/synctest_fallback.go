//go:build !synctest

package mcpscripttest

const hasSynctest = false

// SynctestSupported returns false when synctest is not available
func SynctestSupported() bool {
	return false
}

// RunWithSynctest fallback just runs the function normally
func RunWithSynctest(f func()) {
	f()
}

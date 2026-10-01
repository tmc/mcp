//go:build !synctest

package mcpscripttest

// SynctestSupported returns false when synctest is not available
func SynctestSupported() bool {
	return false
}

// RunWithSynctest fallback just runs the function normally
func RunWithSynctest(f func()) {
	f()
}

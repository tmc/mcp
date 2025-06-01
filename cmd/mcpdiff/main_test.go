package main

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

// TestMCPDiffBasic runs all script tests for mcpdiff
func TestMCPDiffBasic(t *testing.T) {
	// Setup coverage environment automatically
	mcpscripttest.SetupCoverageEnvironment(t)

	// Run all tests in the testdata directory
	mcpscripttest.Test(t, "testdata/*.txt")
}

// TestBasicDiff specifically tests basic diff functionality
func TestBasicDiff(t *testing.T) {
	mcpscripttest.Test(t, "testdata/basic_diff.txt")
}

// TestNotificationDiff specifically tests notification diff functionality
func TestNotificationDiff(t *testing.T) {
	mcpscripttest.Test(t, "testdata/notification_diff_test.txt")
}

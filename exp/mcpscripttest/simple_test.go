package mcpscripttest

import (
	"testing"
	_ "time" // was used in experimental code
)

// TestUltraSimple demonstrates the ultra-simple approach
// Disabled - part of experimental code that was cleaned up
/*
func TestUltraSimple(t *testing.T) {
	t.Log("✅ Ultra-simple in-process testing concept!")
	
	// Define some mock server functions
	servers := map[string]func(){
		"time-server": func() {
			t.Log("Time server running")
			time.Sleep(100 * time.Millisecond) // Simulate some work
		},
		"echo-server": func() {
			t.Log("Echo server running")  
			time.Sleep(100 * time.Millisecond) // Simulate some work
		},
	}
	
	// Start servers, run test, clean up
	TestSimpleInProcess(t, "testdata/simple_*.txt", servers)
	
	t.Log("✅ Test completed successfully!")
}
*/

// TestServerLifecycle tests the server start/stop functionality
// Disabled - part of experimental code that was cleaned up
/*
func TestServerLifecycle(t *testing.T) {
	called := false
	serverFunc := func() {
		called = true
		time.Sleep(50 * time.Millisecond)
	}
	
	// Start server
	StartSimpleServer("test-server", serverFunc)
	
	// Give it time to start
	time.Sleep(10 * time.Millisecond)
	
	if !called {
		t.Error("Server function was not called")
	}
	
	// Stop server
	StopSimpleServer("test-server")
	
	t.Log("✅ Server lifecycle works correctly!")
}
*/

// TestSimpleWithSynctest tests that the API automatically uses synctest when available
// Disabled - part of experimental code that was cleaned up
/*
func TestSimpleWithSynctest(t *testing.T) {
	t.Log("✅ Testing automatic synctest detection!")
	
	// Define some mock server functions
	servers := map[string]func(){
		"time-server": func() {
			if SynctestSupported() {
				t.Log("Time server running with synctest (automatic)")
			} else {
				t.Log("Time server running without synctest")
			}
			time.Sleep(100 * time.Millisecond) // Synthetic or real time
		},
	}
	
	// Same API - automatically uses synctest when available!
	TestSimpleInProcess(t, "testdata/simple_test.txt", servers)
	
	if SynctestSupported() {
		t.Log("✅ Synctest was used automatically!")
	} else {
		t.Log("✅ Regular execution (synctest not available)")
	}
}
*/
//go:build darwin

package mcp

import (
	"log/slog"
	"runtime"
	"testing"
)

func TestGetApplePlatformInfo(t *testing.T) {
	// This test only runs on Darwin/macOS
	if runtime.GOOS != "darwin" {
		t.Skip("Test only runs on Darwin/macOS")
	}

	info := GetApplePlatformInfo()

	if info == nil {
		t.Fatal("GetApplePlatformInfo returned nil")
	}

	// Verify basic fields are populated
	if info.ProcessorCount <= 0 {
		t.Errorf("ProcessorCount should be positive, got %d", info.ProcessorCount)
	}

	if info.SystemVersion == "" {
		t.Error("SystemVersion should not be empty")
	}

	// Architecture-specific checks
	if runtime.GOARCH == "arm64" {
		if !info.IsAppleSilicon {
			t.Error("Should detect Apple Silicon on arm64")
		}
		// On native Apple Silicon, should not be running under Rosetta
		if info.IsRosetta {
			t.Error("Native arm64 should not be running under Rosetta")
		}
	} else if runtime.GOARCH == "amd64" {
		if info.IsAppleSilicon {
			t.Error("Should not detect Apple Silicon on amd64")
		}
		// Note: IsRosetta detection on amd64 could be true or false
		// depending on whether it's native Intel or Rosetta
	}

	t.Logf("Platform info: IsAppleSilicon=%v, IsRosetta=%v, ProcessorCount=%d, SystemVersion=%s",
		info.IsAppleSilicon, info.IsRosetta, info.ProcessorCount, info.SystemVersion)
}

func TestIsAppleSilicon(t *testing.T) {
	result := isAppleSilicon()

	// Should only be true on arm64 Darwin
	expected := runtime.GOARCH == "arm64" && runtime.GOOS == "darwin"

	if result != expected {
		t.Errorf("isAppleSilicon() = %v, expected %v (arch=%s, os=%s)",
			result, expected, runtime.GOARCH, runtime.GOOS)
	}
}

func TestGetOptimalTransportConfig(t *testing.T) {
	config := GetOptimalTransportConfig()

	if config == nil {
		t.Fatal("GetOptimalTransportConfig returned nil")
	}

	// Verify sensible defaults
	if !config.UseKqueue {
		t.Error("UseKqueue should be true on Darwin")
	}

	if !config.UseGrandCentral {
		t.Error("UseGrandCentral should be true on Darwin")
	}

	if config.BufferSize <= 0 {
		t.Errorf("BufferSize should be positive, got %d", config.BufferSize)
	}

	// Buffer size should be reasonable (between 16KB and 256KB)
	if config.BufferSize < 16*1024 || config.BufferSize > 256*1024 {
		t.Errorf("BufferSize %d seems unreasonable", config.BufferSize)
	}

	t.Logf("Transport config: UseKqueue=%v, UseGrandCentral=%v, BufferSize=%d",
		config.UseKqueue, config.UseGrandCentral, config.BufferSize)
}

func TestAppleOptimizedTransport(t *testing.T) {
	// Create a mock base transport
	base := &ReadWriteCloserTransport{
		ReadWriteCloser: &mockReadWriteCloser{},
	}

	transport := NewAppleOptimizedTransport(base)

	if transport == nil {
		t.Fatal("NewAppleOptimizedTransport returned nil")
	}

	if transport.ReadWriteCloserTransport != base {
		t.Error("Base transport not properly wrapped")
	}

	if transport.config == nil {
		t.Error("Transport config not initialized")
	}

	if transport.platformInfo == nil {
		t.Error("Platform info not initialized")
	}
}

func TestAppleOptimizedTransportRead(t *testing.T) {
	mock := &mockReadWriteCloser{
		readData: []byte("Hello, MCP on Apple!"),
	}

	base := &ReadWriteCloserTransport{ReadWriteCloser: mock}
	transport := NewAppleOptimizedTransport(base)

	// Test reading with small buffer
	smallBuf := make([]byte, 10)
	n, err := transport.Read(smallBuf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if n != 10 {
		t.Errorf("Expected to read 10 bytes, got %d", n)
	}

	expected := "Hello, MCP"
	if string(smallBuf) != expected {
		t.Errorf("Expected %q, got %q", expected, string(smallBuf))
	}
}

func TestAppleOptimizedTransportWrite(t *testing.T) {
	mock := &mockReadWriteCloser{}
	base := &ReadWriteCloserTransport{ReadWriteCloser: mock}
	transport := NewAppleOptimizedTransport(base)

	testData := []byte("Test write data for Apple optimized transport")

	n, err := transport.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, got %d", len(testData), n)
	}

	if string(mock.writtenData) != string(testData) {
		t.Errorf("Written data mismatch: expected %q, got %q", string(testData), string(mock.writtenData))
	}
}

func TestAppleOptimizedTransportGetOptimizationInfo(t *testing.T) {
	base := &ReadWriteCloserTransport{
		ReadWriteCloser: &mockReadWriteCloser{},
	}
	transport := NewAppleOptimizedTransport(base)

	info := transport.GetOptimizationInfo()

	if info == nil {
		t.Fatal("GetOptimizationInfo returned nil")
	}

	// Verify required fields
	requiredFields := []string{
		"platform", "isAppleSilicon", "isRosetta", "processorCount",
		"systemVersion", "bufferSize", "useKqueue", "useGrandCentral",
	}

	for _, field := range requiredFields {
		if _, exists := info[field]; !exists {
			t.Errorf("Missing required field: %s", field)
		}
	}

	// Verify platform is correct
	if info["platform"] != "darwin" {
		t.Errorf("Expected platform 'darwin', got %v", info["platform"])
	}

	t.Logf("Optimization info: %+v", info)
}

func TestWithAppleTransport(t *testing.T) {
	base := &ReadWriteCloserTransport{
		ReadWriteCloser: &mockReadWriteCloser{},
	}

	transport := WithAppleTransport(base)

	// On Darwin, should return Apple-optimized transport
	if runtime.GOOS == "darwin" {
		appleTransport, ok := transport.(*AppleOptimizedTransport)
		if !ok {
			t.Error("Expected AppleOptimizedTransport on Darwin")
		} else if appleTransport.ReadWriteCloserTransport != base {
			t.Error("Base transport not properly wrapped")
		}
	} else {
		// On non-Darwin, should return base transport
		if transport != base {
			t.Error("Expected base transport on non-Darwin platforms")
		}
	}
}

func TestApplePerformanceMonitor(t *testing.T) {
	monitor := NewApplePerformanceMonitor()

	if monitor == nil {
		t.Fatal("NewApplePerformanceMonitor returned nil")
	}

	// Should be enabled on Darwin
	if runtime.GOOS == "darwin" && !monitor.enabled {
		t.Error("Monitor should be enabled on Darwin")
	}

	// Test metric recording
	monitor.RecordMetric("test_metric", 42)
	monitor.RecordMetric("test_string", "hello")

	stats := monitor.GetStats()

	if runtime.GOOS == "darwin" {
		if stats == nil {
			t.Error("GetStats should return stats on Darwin")
		} else {
			if stats["test_metric"] != 42 {
				t.Error("Metric not recorded correctly")
			}
			if stats["test_string"] != "hello" {
				t.Error("String metric not recorded correctly")
			}

			// Should include platform info
			if _, exists := stats["platform_info"]; !exists {
				t.Error("Platform info missing from stats")
			}
		}
	} else {
		if stats != nil {
			t.Error("GetStats should return nil on non-Darwin")
		}
	}
}

func TestAppleMemoryOptimizer(t *testing.T) {
	optimizer := NewAppleMemoryOptimizer()

	if optimizer == nil {
		t.Fatal("NewAppleMemoryOptimizer returned nil")
	}

	if optimizer.platformInfo == nil {
		t.Error("Platform info not initialized")
	}

	// Test buffer size optimization
	bufferSize := optimizer.GetOptimalBufferSize()
	if bufferSize <= 0 {
		t.Errorf("Buffer size should be positive, got %d", bufferSize)
	}

	// Should be reasonable size (between 16KB and 256KB)
	if bufferSize < 16*1024 || bufferSize > 256*1024 {
		t.Errorf("Buffer size %d seems unreasonable", bufferSize)
	}

	// Test concurrency optimization
	concurrency := optimizer.GetOptimalConcurrency()
	if concurrency <= 0 {
		t.Errorf("Concurrency should be positive, got %d", concurrency)
	}

	// Should be related to processor count
	if concurrency > runtime.NumCPU()*4 || concurrency < 1 {
		t.Errorf("Concurrency %d seems unreasonable for %d CPUs", concurrency, runtime.NumCPU())
	}

	// Test memory mapping decisions
	smallSize := int64(32 * 1024)       // 32KB
	largeSize := int64(2 * 1024 * 1024) // 2MB

	smallMapping := optimizer.ShouldUseMemoryMapping(smallSize)
	largeMapping := optimizer.ShouldUseMemoryMapping(largeSize)

	// Large files should generally use memory mapping
	if !largeMapping {
		t.Error("Large files should use memory mapping")
	}

	t.Logf("Memory optimizer: BufferSize=%d, Concurrency=%d, SmallMapping=%v, LargeMapping=%v",
		bufferSize, concurrency, smallMapping, largeMapping)
}

func TestWithAppleOptimizations(t *testing.T) {
	// Test that the option can be created without panic
	option := WithAppleOptimizations()
	if option == nil {
		t.Error("WithAppleOptimizations returned nil")
	}

	// Test applying the option to a server
	server := NewServer("test-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))
	originalServer := *server // Copy for comparison

	// Apply the option
	option(server)

	// The option should not break the server
	if server == nil {
		t.Error("Server became nil after applying Apple optimizations")
	}

	// Basic server functionality should still work
	if server.name != originalServer.name {
		t.Error("Server name changed after applying optimizations")
	}

	if server.version != originalServer.version {
		t.Error("Server version changed after applying optimizations")
	}
}

// Mock ReadWriteCloser for testing
type mockReadWriteCloser struct {
	readData    []byte
	readPos     int
	writtenData []byte
	closed      bool
}

func (m *mockReadWriteCloser) Read(p []byte) (n int, err error) {
	if m.readPos >= len(m.readData) {
		return 0, nil // EOF
	}

	n = copy(p, m.readData[m.readPos:])
	m.readPos += n
	return n, nil
}

func (m *mockReadWriteCloser) Write(p []byte) (n int, err error) {
	m.writtenData = append(m.writtenData, p...)
	return len(p), nil
}

func (m *mockReadWriteCloser) Close() error {
	m.closed = true
	return nil
}

// Benchmark Apple-specific optimizations
func BenchmarkAppleOptimizedTransportRead(b *testing.B) {
	if runtime.GOOS != "darwin" {
		b.Skip("Benchmark only runs on Darwin")
	}

	data := make([]byte, 64*1024) // 64KB test data
	for i := range data {
		data[i] = byte(i % 256)
	}

	mock := &mockReadWriteCloser{readData: data}
	base := &ReadWriteCloserTransport{ReadWriteCloser: mock}
	transport := NewAppleOptimizedTransport(base)

	buf := make([]byte, 4096)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.readPos = 0 // Reset for each iteration
		for mock.readPos < len(data) {
			_, err := transport.Read(buf)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkAppleOptimizedTransportWrite(b *testing.B) {
	if runtime.GOOS != "darwin" {
		b.Skip("Benchmark only runs on Darwin")
	}

	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(i % 256)
	}

	mock := &mockReadWriteCloser{}
	base := &ReadWriteCloserTransport{ReadWriteCloser: mock}
	transport := NewAppleOptimizedTransport(base)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.writtenData = mock.writtenData[:0] // Reset buffer
		_, err := transport.Write(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

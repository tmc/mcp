package mcp

// mockReadWriteCloser for testing - shared across test files
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

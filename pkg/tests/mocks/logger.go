package mocks

import "strings"

// MockLogger is a struct, which allows mocking of the logging facility.
type MockLogger struct {
	// LogLines is the string list, where we accumulate log lines.
	LogLines []string
}

// Info emits a log line.
func (m *MockLogger) Info(args ...interface{}) {
	var stringArgs []string
	for _, arg := range args {
		stringArgs = append(stringArgs, arg.(string))
	}
	m.LogLines = append(m.LogLines, strings.Join(stringArgs, "_"))
}

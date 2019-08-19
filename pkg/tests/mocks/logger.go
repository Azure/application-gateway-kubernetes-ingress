package mocks

import "strings"

type MockLogger struct {
	LogLines []string
}

func (m *MockLogger) Info(args ...interface{}) {
	var stringArgs []string
	for _, arg := range args {
		stringArgs = append(stringArgs, arg.(string))
	}
	m.LogLines = append(m.LogLines, strings.Join(stringArgs, "_"))
}

package lxc

import (
	"errors"
	"strings"
)

// MockExecutor is a mock LXC executor for testing
type MockExecutor struct {
	// Calls records all calls made
	Calls []MockCall

	// Responses maps command patterns to responses
	Responses map[string]MockResponse

	// DefaultResponse is returned when no matching response is found
	DefaultResponse MockResponse

	// Callbacks maps command patterns to functions called when the command is executed
	// The callback receives the full args slice
	Callbacks map[string]func(args []string)
}

// MockCall represents a single call to the executor
type MockCall struct {
	Args []string
}

// MockResponse represents a mock response
type MockResponse struct {
	Output []byte
	Err    error
}

// NewMockExecutor creates a new mock executor
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		Calls:     []MockCall{},
		Responses: make(map[string]MockResponse),
		Callbacks: make(map[string]func(args []string)),
	}
}

// Run implements Executor
func (m *MockExecutor) Run(args ...string) ([]byte, error) {
	m.Calls = append(m.Calls, MockCall{Args: args})
	return m.getResponse(args)
}

// RunCombined implements Executor
func (m *MockExecutor) RunCombined(args ...string) ([]byte, error) {
	m.Calls = append(m.Calls, MockCall{Args: args})
	return m.getResponse(args)
}

func (m *MockExecutor) getResponse(args []string) ([]byte, error) {
	key := strings.Join(args, " ")

	// Execute callbacks (try exact match first, then prefix match)
	if cb, ok := m.Callbacks[key]; ok {
		cb(args)
	} else {
		for pattern, cb := range m.Callbacks {
			if strings.HasPrefix(key, pattern) {
				cb(args)
				break
			}
		}
	}

	// Try exact match first
	if resp, ok := m.Responses[key]; ok {
		return resp.Output, resp.Err
	}

	// Try prefix match
	for pattern, resp := range m.Responses {
		if strings.HasPrefix(key, pattern) {
			return resp.Output, resp.Err
		}
	}

	// Return default
	return m.DefaultResponse.Output, m.DefaultResponse.Err
}

// SetResponse sets a response for a command pattern
func (m *MockExecutor) SetResponse(pattern string, output []byte, err error) {
	m.Responses[pattern] = MockResponse{Output: output, Err: err}
}

// SetError sets an error response for a command pattern
func (m *MockExecutor) SetError(pattern string, errMsg string) {
	m.Responses[pattern] = MockResponse{Err: errors.New(errMsg)}
}

// SetOutput sets a successful output for a command pattern
func (m *MockExecutor) SetOutput(pattern string, output string) {
	m.Responses[pattern] = MockResponse{Output: []byte(output)}
}

// SetCallback sets a callback function for a command pattern
// The callback is called when the command is executed, before returning the response
func (m *MockExecutor) SetCallback(pattern string, cb func(args []string)) {
	m.Callbacks[pattern] = cb
}

// Reset clears all calls and responses
func (m *MockExecutor) Reset() {
	m.Calls = []MockCall{}
	m.Responses = make(map[string]MockResponse)
	m.Callbacks = make(map[string]func(args []string))
	m.DefaultResponse = MockResponse{}
}

// CallCount returns the number of calls made
func (m *MockExecutor) CallCount() int {
	return len(m.Calls)
}

// LastCall returns the last call made
func (m *MockExecutor) LastCall() MockCall {
	if len(m.Calls) == 0 {
		return MockCall{}
	}
	return m.Calls[len(m.Calls)-1]
}

// HasCall checks if a call with the given args was made
func (m *MockExecutor) HasCall(args ...string) bool {
	target := strings.Join(args, " ")
	for _, call := range m.Calls {
		if strings.Join(call.Args, " ") == target {
			return true
		}
	}
	return false
}

// HasCallPrefix checks if a call starting with the given args was made
func (m *MockExecutor) HasCallPrefix(args ...string) bool {
	target := strings.Join(args, " ")
	for _, call := range m.Calls {
		callStr := strings.Join(call.Args, " ")
		if strings.HasPrefix(callStr, target) {
			return true
		}
	}
	return false
}

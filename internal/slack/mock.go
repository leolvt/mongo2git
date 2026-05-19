package slack

// MockNotifier records calls for test assertions.
type MockNotifier struct {
	SuccessCalled bool
	FailureCalled bool
	LastSuccess   bool
	LastTimestamp string
	LastDocCount  int
	LastDetail    string
}

func (m *MockNotifier) Notify(success bool, timestamp string, docCount int, detail string) {
	if success {
		m.SuccessCalled = true
	} else {
		m.FailureCalled = true
	}
	m.LastSuccess = success
	m.LastTimestamp = timestamp
	m.LastDocCount = docCount
	m.LastDetail = detail
}

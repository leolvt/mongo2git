package git

// MockRepo is a RepoOps that returns pre-configured responses, for use in tests.
type MockRepo struct {
	PrepareErr      error
	Changed         bool
	Err             error
	CommitAndPushFn func(timestamp string, docCount int) (bool, error)
}

func (m *MockRepo) Prepare() error { return m.PrepareErr }

func (m *MockRepo) CommitAndPush(timestamp string, docCount int) (bool, error) {
	if m.CommitAndPushFn != nil {
		return m.CommitAndPushFn(timestamp, docCount)
	}
	return m.Changed, m.Err
}

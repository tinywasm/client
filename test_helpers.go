package client

type testDatabase struct {
	data map[string]string
}

func (s *testDatabase) Get(key string) (string, error) {
	if val, ok := s.data[key]; ok {
		return val, nil
	}
	return "", nil
}

func (s *testDatabase) Set(key, value string) error {
	s.data[key] = value
	return nil
}

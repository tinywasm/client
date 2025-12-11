package client

type testStore struct {
	data map[string]string
}

func (s *testStore) Get(key string) (string, error) {
	if val, ok := s.data[key]; ok {
		return val, nil
	}
	return "", nil
}

func (s *testStore) Set(key, value string) error {
	s.data[key] = value
	return nil
}

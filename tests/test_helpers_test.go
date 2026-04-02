package client_test


type TestDatabase struct {
	data map[string]string
}

func (s *TestDatabase) Get(key string) (string, error) {
	if s.data == nil {
		return "", nil
	}
	if val, ok := s.data[key]; ok {
		return val, nil
	}
	return "", nil
}

func (s *TestDatabase) Set(key, value string) error {
	if s.data == nil {
		s.data = make(map[string]string)
	}
	s.data[key] = value
	return nil
}

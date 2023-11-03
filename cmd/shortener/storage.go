package main

type Storage map[string]string

func (s Storage) Get(key string) (string, bool) {
	value, ok := s[key]
	return value, ok
}

func (s Storage) Put(key, value string) {
	s[key] = value
}

func (s Storage) KeyByValue(value string) (string, bool) {
	for k, v := range s {
		if v == value {
			return k, true
		}
	}
	return "", false
}

func (s Storage) Clear() {
	for k := range s {
		delete(s, k)
	}
}

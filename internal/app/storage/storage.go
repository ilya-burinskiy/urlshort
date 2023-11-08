package storage

type Storage map[string]string

func (storage Storage) Get(key string) (string, bool) {
	value, ok := storage[key]
	return value, ok
}

func (storage Storage) Put(key, value string) {
	storage[key] = value
}

func (storage Storage) KeyByValue(value string) (string, bool) {
	for k, v := range storage {
		if v == value {
			return k, true
		}
	}
	return "", false
}

func (storage Storage) Clear() {
	for k := range storage {
		delete(storage, k)
	}
}

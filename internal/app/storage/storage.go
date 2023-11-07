package storage

type Storage map[string]string

var storage = make(Storage)

func Get(key string) (string, bool) {
	value, ok := storage[key]
	return value, ok
}

func Put(key, value string) {
	storage[key] = value
}

func KeyByValue(value string) (string, bool) {
	for k, v := range storage {
		if v == value {
			return k, true
		}
	}
	return "", false
}

func Clear() {
	for k := range storage {
		delete(storage, k)
	}
}

package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type MapStorage struct {
	m                 map[string]string
	persistentStorage PersistentStorage
}

func NewMapStorage(ps PersistentStorage) MapStorage {
	return MapStorage{m: make(map[string]string), persistentStorage: ps}
}

func (ms MapStorage) Put(key, val string) {
	ms.m[key] = val
}

func (ms MapStorage) Get(key string) (string, bool) {
	val, ok := ms.m[key]
	return val, ok
}

func (ms MapStorage) Clear() {
	for k := range ms.m {
		delete(ms.m, k)
	}
}

func (ms MapStorage) KeyByValue(value string) (string, bool) {
	for k, v := range ms.m {
		if v == value {
			return k, true
		}
	}
	return "", false
}

func (ms MapStorage) Dump() error {
	if ms.persistentStorage != nil {
		return ms.persistentStorage.Dump(ms)
	}
	return nil
}

func (ms MapStorage) Restore() error {
	if ms.persistentStorage != nil {
		return ms.persistentStorage.Restore(ms)
	}
	return nil
}

type PersistentStorage interface {
	Dump(ms MapStorage) error
	Restore(ms MapStorage) error
}

type FileStorage struct {
	filePath string
}

type record struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

func NewFileStorage(filePath string) FileStorage {
	return FileStorage{filePath: filePath}
}

func (fs FileStorage) Restore(ms MapStorage) error {
	file, err := os.OpenFile(fs.filePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("could not load data from file: %s", err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var r record
		data := scanner.Bytes()
		err = json.Unmarshal(data, &r)
		if err != nil {
			continue
		}

		ms.Put(r.Key, r.Val)
	}

	return scanner.Err()
}

func (fs FileStorage) Dump(ms MapStorage) error {
	file, err := os.OpenFile(fs.filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("could not dump storage: %s", err)
	}

	encoder := json.NewEncoder(file)
	// NOTE: maybe define some Iter method for MapStorage
	for k, v := range ms.m {
		encoder.Encode(record{Key: k, Val: v})
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("could not dump storage: %s", err)
	}

	return nil
}

type Storage struct {
	m        map[string]string
	filePath string
}

type record struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

func New(storagePath string) Storage {
	return Storage{
		m:        map[string]string{},
		filePath: storagePath,
	}
}

func (storage Storage) Get(key string) (string, bool) {
	value, ok := storage.m[key]
	return value, ok
}

func (storage Storage) Put(key, value string) (err error) {
	_, exists := storage.m[key]
	if exists {
		return nil
	}
	storage.m[key] = value

	return nil
}

func (storage Storage) KeyByValue(value string) (string, bool) {
	for k, v := range storage.m {
		if v == value {
			return k, true
		}
	}
	return "", false
}

func (storage Storage) Clear() error {
	for k := range storage.m {
		delete(storage.m, k)
	}

	return nil
}

func (storage Storage) Load() error {
	file, err := os.OpenFile(storage.filePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("could not load data from file: %s", err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var r record
		data := scanner.Bytes()
		err = json.Unmarshal(data, &r)
		if err != nil {
			continue
		}

		storage.m[r.Key] = r.Val
	}

	return scanner.Err()
}

func (storage *Storage) Dump() error {
	file, err := os.OpenFile(storage.filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("could not dump storage: %s", err)
	}

	encoder := json.NewEncoder(file)
	for k, v := range storage.m {
		encoder.Encode(record{Key: k, Val: v})
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("could not dump storage: %s", err)
	}

	return nil
}

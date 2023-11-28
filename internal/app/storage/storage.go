package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

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

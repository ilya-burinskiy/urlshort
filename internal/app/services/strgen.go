package services

import (
	"crypto/rand"
	"encoding/hex"
)

type StdRandHexStringGenerator struct{}

func (randGen StdRandHexStringGenerator) Call(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

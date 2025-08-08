package services

import (
	"crypto/rand"
	"encoding/hex"
)

// RandHexStrGenerator
type RandHexStrGenerator struct{}

// Call
func (randGen RandHexStrGenerator) Gen(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

package logrecord

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type LogRecord struct {
	ID         string
	Timestamp  time.Time
	Message    string
	Service    string
	Status     string
	Attributes map[string]any
	Raw        string
}

func NewID() (string, error) {
	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate log record id: %w", err)
	}

	return hex.EncodeToString(b), nil
}

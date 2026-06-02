package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const tokenBytes = 32

func NewToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func SaveToken(path, token string) error {
	if token == "" {
		return errors.New("token is empty")
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("mkdir auth dir: %w", err)
		}
		_ = os.Chmod(dir, 0o700)
	}
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return fmt.Errorf("write token: %w", err)
	}
	return os.Chmod(path, 0o600)
}

func LoadToken(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read token: %w", err)
	}
	token := string(raw)
	for len(token) > 0 && (token[len(token)-1] == '\n' || token[len(token)-1] == ' ' || token[len(token)-1] == '\r') {
		token = token[:len(token)-1]
	}
	if token == "" {
		return "", errors.New("token file is empty")
	}
	return token, nil
}

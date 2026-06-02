package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
)

func HashToken(token string) (string, error) {
	if token == "" {
		return "", errors.New("token is empty")
	}
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

func VerifyToken(hash, token string) bool {
	want, err := base64.RawURLEncoding.DecodeString(hash)
	if err != nil {
		return false
	}
	sum := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(want, sum[:]) == 1
}

package auth

import "testing"

func TestTokenConstantTimeCompare(t *testing.T) {
	hash, err := HashToken("super-secret-token")
	if err != nil {
		t.Fatalf("HashToken: %v", err)
	}
	if !VerifyToken(hash, "super-secret-token") {
		t.Errorf("VerifyToken with correct secret should pass")
	}
	if VerifyToken(hash, "wrong-token") {
		t.Errorf("VerifyToken with wrong secret should fail")
	}
}

func TestHashTokenRejectsEmpty(t *testing.T) {
	if _, err := HashToken(""); err == nil {
		t.Errorf("HashToken(\"\") should fail")
	}
}

func TestHashTokenFormat(t *testing.T) {
	h, err := HashToken("abc")
	if err != nil {
		t.Fatalf("HashToken: %v", err)
	}
	// base64.RawURLEncoding(SHA-256) = 43 字符
	if len(h) != 43 {
		t.Errorf("hash length = %d, want 43", len(h))
	}
}

func TestVerifyTokenRejectsInvalidBase64(t *testing.T) {
	// malformed hash (not valid base64url) should return false, not panic
	if VerifyToken("!!!not-base64!!!", "any-token") {
		t.Errorf("VerifyToken with malformed hash should return false")
	}
}

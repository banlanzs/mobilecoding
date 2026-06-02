package auth

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadOrCreateCA_New(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.crt")
	ca, err := LoadOrCreateCA(path)
	if err != nil {
		t.Fatalf("LoadOrCreateCA: %v", err)
	}
	if ca.Certificate == nil {
		t.Fatal("CA cert should not be nil")
	}
	if !ca.Certificate.IsCA {
		t.Error("cert should be a CA")
	}
	if !ca.Certificate.NotAfter.After(time.Now().Add(365 * 24 * time.Hour)) {
		t.Error("CA should be valid for at least 1 year")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("CA file should exist: %v", err)
	}
}

func TestLoadOrCreateCA_Existing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.crt")
	ca1, err := LoadOrCreateCA(path)
	if err != nil {
		t.Fatalf("first LoadOrCreateCA: %v", err)
	}
	ca2, err := LoadOrCreateCA(path)
	if err != nil {
		t.Fatalf("second LoadOrCreateCA: %v", err)
	}
	if !ca1.Certificate.Equal(ca2.Certificate) {
		t.Errorf("second call should load existing cert, not regenerate")
	}
}

func TestCAFileFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.crt")
	_, err := LoadOrCreateCA(path)
	if err != nil {
		t.Fatalf("LoadOrCreateCA: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ca: %v", err)
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		t.Fatalf("CA file should be PEM-encoded")
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		t.Errorf("PEM block should be valid x509 cert: %v", err)
	}
}

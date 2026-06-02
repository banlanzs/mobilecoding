package auth

import (
	"testing"
)

func TestGenerateQRCodeASCII(t *testing.T) {
	qr, err := GenerateQRCodeASCII("https://192.168.1.10:8443/?token=test123")
	if err != nil {
		t.Fatalf("GenerateQRCodeASCII: %v", err)
	}
	if len(qr) == 0 {
		t.Error("QR code should not be empty")
	}
	t.Logf("QR Code:\n%s", qr)
}
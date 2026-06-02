package auth

import (
	"fmt"
	"net"
	"strings"

	"github.com/skip2/go-qrcode"
)

// GenerateQRCodeASCII 生成 ASCII art 二维码。
func GenerateQRCodeASCII(content string) (string, error) {
	qr, err := qrcode.New(content, qrcode.Low)
	if err != nil {
		return "", fmt.Errorf("generate qr: %w", err)
	}
	return qr.ToSmallString(false), nil
}

// PrintQRCode 在终端打印二维码。
func PrintQRCode(content string) error {
	qr, err := GenerateQRCodeASCII(content)
	if err != nil {
		return err
	}
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("Scan QR Code to connect:")
	fmt.Println(qr)
	fmt.Println(strings.Repeat("=", 50))
	return nil
}

// GetLocalIP 返回本机第一个非回环的 IPv4 地址；若失败则返回 "127.0.0.1"。
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil || ip.IsLoopback() {
			continue
		}
		ip = ip.To4()
		if ip == nil {
			continue
		}
		return ip.String()
	}
	return "127.0.0.1"
}
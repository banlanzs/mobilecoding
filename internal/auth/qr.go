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

// GetLocalIP 返回本机最适合对外服务的 IPv4 地址。
// 优先级：局域网 A 段 (10.x.x.x) > 局域网 B 段 (172.16-31.x.x) > 局域网 C 段 (192.168.x.x) > 其他非回环 > 127.0.0.1。
// 自动跳过 link-local (169.254.x.x) 和已知虚拟网卡地址。
func GetLocalIP() string {
	ips := GetAllLocalIPs()
	if len(ips) == 0 {
		return "127.0.0.1"
	}
	// 按优先级筛选局域网 IP，跳过 link-local 和虚拟网卡
	for _, prefix := range []string{"10.", "172.", "192.168."} {
		for _, ip := range ips {
			if !strings.HasPrefix(ip, prefix) {
				continue
			}
			// 172.x 需要额外检查 16-31 范围
			if prefix == "172." {
				parts := strings.Split(ip, ".")
				if len(parts) >= 2 {
					var octet int
					fmt.Sscanf(parts[1], "%d", &octet)
					if octet < 16 || octet > 31 {
						continue
					}
				}
			}
			// 192.168.56.x 是 VirtualBox Host-Only，优先级降低
			if strings.HasPrefix(ip, "192.168.56.") {
				continue
			}
			return ip
		}
	}
	// 退回：跳过 link-local (169.254.x.x) 后的第一个
	for _, ip := range ips {
		if !strings.HasPrefix(ip, "169.254.") {
			return ip
		}
	}
	// 全是 link-local，返回第一个
	return ips[0]
}

// GetAllLocalIPs 返回本机所有非回环的 IPv4 地址列表。
func GetAllLocalIPs() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	var ips []string
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
		ips = append(ips, ip.String())
	}
	return ips
}
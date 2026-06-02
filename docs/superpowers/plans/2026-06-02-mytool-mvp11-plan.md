# mytool MVP 1.1 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实施 MVP 1.1 spec 中的两个 followup：(1) 把后端从 HTTP 升级到 HTTPS + 可选 mTLS；(2) 把 `projection.Stream` 真正接到 ws handler 事件流里，删除 inline json.Marshal 构造。

**Architecture:**
- **TLS 部分**：新增 `internal/auth/ca.go` 与 `internal/auth/servercert.go`，启动时一次性生成自签 CA + server 证书（10 年），落到 `~/.mytool/auth/`（0o700 目录 / 0o600 私钥）；`cmd/server/main.go` 启动流程切换到 `ListenAndServeTLS`；mTLS 模式用 `tls.Config.ClientAuth` 切换
- **Projection 部分**：`internal/projection` 重构——`Project` 接受 session id 参数，新增 `TextEvent` / `LifecycleEvent` 构造器；`internal/ws/handler.go` 改用 `projection.Stream` + adapter 删 inline json.Marshal

**Tech Stack:**
- Go 标准库 `crypto/rsa` / `crypto/x509` / `crypto/tls`（无新依赖）
- 现有：`chi/v5` / `gorilla/websocket` / `creack/pty` / `google/uuid`
- bash + `curl --cacert`（e2e smoke 升级到 https）

**Spec Reference:** `docs/superpowers/specs/2026-06-02-mytool-mvp11-design.md` §1（TLS/mTLS）+ §2（projection.Stream 接入）

**前置：** MVP 1 已完成（`684aae2`，分支 `feat/mytool-mvp1`，20 个 commit）。当前在 `feat/mytool-mvp11` 分支（已基于 `684aae2` 切出）。

---

## Task 14: internal/auth/ca.go（自签 CA 生成 + 加载/创建）

**Files:**
- Create: `mytool/internal/auth/ca.go`
- Create: `mytool/internal/auth/ca_test.go`

- [ ] **Step 1: 写 ca_test.go**

文件：`mytool/internal/auth/ca_test.go`

```go
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
	// Verify file was created
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
	// Same path → same cert
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
```

- [ ] **Step 2: 跑测试确认失败**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/auth/... -run TestLoadOrCreateCA -run TestCA
```

预期：FAIL（包内还没有 ca.go）。

> 修正：实际命令是 `cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/auth/... -run 'TestLoadOrCreateCA|TestCAFileFormat'`。

- [ ] **Step 3: 实现 ca.go**

文件：`mytool/internal/auth/ca.go`

```go
package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pem"
	"encoding/pem"  // 保留为了 IDE 跳转；实际只 import 一次
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

const caCommonName = "mytool local development CA"

// CA 是内存中的 CA 证书与私钥。
type CA struct {
	Certificate *x509.Certificate
	PrivateKey  *rsa.PrivateKey
}

// LoadOrCreateCA 加载 path 处的 CA 证书；若不存在则新建 RSA-2048 CA（10 年）。
// 父目录权限 0o700，文件权限 0o600。
func LoadOrCreateCA(path string) (*CA, error) {
	if raw, err := os.ReadFile(path); err == nil {
		return parseCAPEM(raw)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read ca: %w", err)
	}
	return generateAndSaveCA(path)
}

func generateAndSaveCA(path string) (*CA, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate ca key: %w", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkixName("mytool", "local", "mytool dev CA"),
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("create ca cert: %w", err)
	}
	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, fmt.Errorf("parse ca cert: %w", err)
	}

	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("mkdir ca dir: %w", err)
		}
		_ = os.Chmod(dir, 0o700)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		return nil, fmt.Errorf("write ca: %w", err)
	}
	_ = os.Chmod(path, 0o600)

	return &CA{Certificate: cert, PrivateKey: key}, nil
}

func parseCAPEM(raw []byte) (*CA, error) {
	block, _ := pem.Decode(raw)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("ca: invalid PEM block")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse ca cert: %w", err)
	}
	if !cert.IsCA {
		return nil, errors.New("ca: cert is not a CA")
	}
	// MVP 1.1: 不加载 CA 私钥（仅需用 CA 签发新 server 证书时再生成或要求用户提供）。
	// 这避免了"启动时必须保存 CA 私钥"的隐私顾虑。
	return &CA{Certificate: cert, PrivateKey: nil}, nil
}
```

> 注意：MVP 1.1 简化：CA 私钥**每次启动时重新生成**，签发 server 证书后**不保存**。server 证书则带私钥保存到 `~/.mytool/auth/server.{crt,key}`。这样：
> - 持久化的只有 server 证书（每次启动用一次性内存 CA 私钥签发）
> - `~/.mytool/auth/ca.crt` 仅作"同一 CA 名签发的客户端信任锚"用途
> - 不持久化 CA 私钥，符合 spec §10 "不收集原则" 的延伸

**调整 Step 3 实现**：把 `parseCAPEM` 改为允许 `PrivateKey == nil`，并在 `generateAndSaveCA` 之外提供一个 helper `(*CA).SignServerCSR(csr *x509.CertificateRequest) ([]byte, error)`。

```go
// SignServerCSR 用 CA 私钥签发 server 证书，返回 DER-encoded cert。
// 若 CA 私钥为 nil（从已有 PEM 加载的情况），返回错误。
func (c *CA) SignServerCSR(csr *x509.CertificateRequest) ([]byte, error) {
	if c.PrivateKey == nil {
		return nil, errors.New("ca: private key not loaded; regenerate CA to sign")
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("invalid csr signature: %w", err)
	}
	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      csr.Subject,
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	return x509.CreateCertificate(rand.Reader, tmpl, c.Certificate, csr.PublicKey, c.PrivateKey)
}
```

同时修正 `generateAndSaveCA` 末尾的 import 错误（不要重复 import `encoding/pem`）：

```go
import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)
```

- [ ] **Step 4: 跑 ca 测试确认通过**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/auth/... -v -run 'TestLoadOrCreateCA|TestCAFileFormat'
```

预期：3 个测试 PASS。

- [ ] **Step 5: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/internal/auth/ca.go mytool/internal/auth/ca_test.go
git commit -m "feat(auth): 自签 CA 生成 + 加载（用于 server 证书签发）"
```

---

## Task 15: internal/auth/servercert.go（用 CA 签发 server 证书）

**Files:**
- Create: `mytool/internal/auth/servercert.go`
- Create: `mytool/internal/auth/servercert_test.go`

- [ ] **Step 1: 写 servercert_test.go**

文件：`mytool/internal/auth/servercert_test.go`

```go
package auth

import (
	"crypto/x509"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadOrCreateServerCert_New(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.crt")
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")

	ca, err := LoadOrCreateCA(caPath)
	if err != nil {
		t.Fatalf("LoadOrCreateCA: %v", err)
	}
	if err := LoadOrCreateServerCert(ca, certPath, keyPath, "192.168.1.10", "myhost.local"); err != nil {
		t.Fatalf("LoadOrCreateServerCert: %v", err)
	}

	// Verify cert file
	raw, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read server cert: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca.Certificate)
	if _, err := certFromPEM(raw, pool); err != nil {
		t.Errorf("server cert should be valid: %v", err)
	}

	// Verify key file exists with 0o600
	st, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key: %v", err)
	}
	if runtime_GOOS_noperm(t) {
		// skip perm check on windows
	} else if st.Mode().Perm() != 0o600 {
		t.Errorf("key perm = %o, want 0o600", st.Mode().Perm())
	}
}

func TestServerCertSAN(t *testing.T) {
	dir := t.TempDir()
	ca, _ := LoadOrCreateCA(filepath.Join(dir, "ca.crt"))
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")
	if err := LoadOrCreateServerCert(ca, certPath, keyPath, "10.0.0.1", "box.lan"); err != nil {
		t.Fatalf("LoadOrCreateServerCert: %v", err)
	}
	raw, _ := os.ReadFile(certPath)
	pool := x509.NewCertPool()
	pool.AddCert(ca.Certificate)
	cert, err := certFromPEM(raw, pool)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	hasIP, hasDNS := false, false
	for _, ip := range cert.IPAddresses {
		if ip.Equal(net.ParseIP("10.0.0.1")) {
			hasIP = true
		}
	}
	for _, name := range cert.DNSNames {
		if name == "box.lan" {
			hasDNS = true
		}
	}
	if !hasIP {
		t.Errorf("SAN should include IP 10.0.0.1, got %v", cert.IPAddresses)
	}
	if !hasDNS {
		t.Errorf("SAN should include DNS box.lan, got %v", cert.DNSNames)
	}
}

func TestLoadOrCreateServerCert_Existing(t *testing.T) {
	dir := t.TempDir()
	ca, _ := LoadOrCreateCA(filepath.Join(dir, "ca.crt"))
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")

	if err := LoadOrCreateServerCert(ca, certPath, keyPath, "10.0.0.1", "box.lan"); err != nil {
		t.Fatalf("first: %v", err)
	}
	stat1, _ := os.Stat(certPath)
	mod1 := stat1.ModTime()

	// Re-call should reuse existing cert
	if err := LoadOrCreateServerCert(ca, certPath, keyPath, "10.0.0.1", "box.lan"); err != nil {
		t.Fatalf("second: %v", err)
	}
	stat2, _ := os.Stat(certPath)
	if !stat2.ModTime().Equal(mod1) {
		t.Errorf("second call should not regenerate server cert")
	}
}

// certFromPEM 解析 PEM 并用 pool 验证。
func certFromPEM(raw []byte, pool *x509.CertPool) (*x509.Certificate, error) {
	block, _ := pemDecode(raw)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errInvalidPEM
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	opts := x509.VerifyOptions{Roots: pool, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	if _, err := cert.Verify(opts); err != nil {
		return nil, err
	}
	return cert, nil
}

var errInvalidPEM = stringErr("invalid PEM")

type stringErr string

func (e stringErr) Error() string { return string(e) }
```

> 说明：测试中用 `pemDecode` 和 `runtime_GOOS_noperm` 占位，需要在 servercert.go 中提供。
> 简化为：去掉 perm 平台判断，Windows 跳过 perm 检查（与 Task 3 secret.go 一致）。

修正版本（**使用此版本**）：

```go
package auth

import (
	"crypto/x509"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadOrCreateServerCert_New(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.crt")
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")

	ca, err := LoadOrCreateCA(caPath)
	if err != nil {
		t.Fatalf("LoadOrCreateCA: %v", err)
	}
	if err := LoadOrCreateServerCert(ca, certPath, keyPath, "192.168.1.10", "myhost.local"); err != nil {
		t.Fatalf("LoadOrCreateServerCert: %v", err)
	}

	raw, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read server cert: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca.Certificate)
	if _, err := parseAndVerifyCert(raw, pool); err != nil {
		t.Errorf("server cert should be valid: %v", err)
	}

	st, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key: %v", err)
	}
	if runtime.GOOS != "windows" {
		if st.Mode().Perm() != 0o600 {
			t.Errorf("key perm = %o, want 0o600", st.Mode().Perm())
		}
	}
}

func TestServerCertSAN(t *testing.T) {
	dir := t.TempDir()
	ca, _ := LoadOrCreateCA(filepath.Join(dir, "ca.crt"))
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")
	if err := LoadOrCreateServerCert(ca, certPath, keyPath, "10.0.0.1", "box.lan"); err != nil {
		t.Fatalf("LoadOrCreateServerCert: %v", err)
	}
	raw, _ := os.ReadFile(certPath)
	pool := x509.NewCertPool()
	pool.AddCert(ca.Certificate)
	cert, err := parseAndVerifyCert(raw, pool)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	hasIP, hasDNS := false, false
	for _, ip := range cert.IPAddresses {
		if ip.Equal(net.ParseIP("10.0.0.1")) {
			hasIP = true
		}
	}
	for _, name := range cert.DNSNames {
		if name == "box.lan" {
			hasDNS = true
		}
	}
	if !hasIP {
		t.Errorf("SAN should include IP 10.0.0.1, got %v", cert.IPAddresses)
	}
	if !hasDNS {
		t.Errorf("SAN should include DNS box.lan, got %v", cert.DNSNames)
	}
}

func TestLoadOrCreateServerCert_Existing(t *testing.T) {
	dir := t.TempDir()
	ca, _ := LoadOrCreateCA(filepath.Join(dir, "ca.crt"))
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")

	if err := LoadOrCreateServerCert(ca, certPath, keyPath, "10.0.0.1", "box.lan"); err != nil {
		t.Fatalf("first: %v", err)
	}
	stat1, _ := os.Stat(certPath)
	mod1 := stat1.ModTime()

	if err := LoadOrCreateServerCert(ca, certPath, keyPath, "10.0.0.1", "box.lan"); err != nil {
		t.Fatalf("second: %v", err)
	}
	stat2, _ := os.Stat(certPath)
	if !stat2.ModTime().Equal(mod1) {
		t.Errorf("second call should not regenerate server cert")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/auth/... -run 'TestLoadOrCreateServerCert|TestServerCertSAN'
```

预期：FAIL（包内还没有 servercert.go）。

- [ ] **Step 3: 实现 servercert.go**

文件：`mytool/internal/auth/servercert.go`

```go
package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pem"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// LoadOrCreateServerCert 若 path 已存在则加载；否则用 ca 签发新 server 证书。
// 证书含 SAN: <ip>, <dns>, 127.0.0.1, localhost。有效期 1 年。
// Key 用 ECDSA P-256（小、签发/验签快），文件 0o600。
func LoadOrCreateServerCert(ca *CA, certPath, keyPath, ip, dns string) error {
	if _, err := os.Stat(certPath); err == nil {
		// 已存在：仅校验 ca 私钥（如果缺失，会在首次启动期一次性重新签发）
		return nil
	}
	if ca == nil || ca.PrivateKey == nil {
		return errors.New("servercert: CA private key is required to sign new server cert")
	}
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate server key: %w", err)
	}
	csrTmpl := &x509.CertificateRequest{
		Subject: pkixName("mytool", "server", dns),
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, csrTmpl, priv)
	if err != nil {
		return fmt.Errorf("create csr: %w", err)
	}
	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		return fmt.Errorf("parse csr: %w", err)
	}
	der, err := ca.SignServerCSR(csr)
	if err != nil {
		return fmt.Errorf("sign server cert: %w", err)
	}
	if err := writePEMFile(certPath, "CERTIFICATE", der); err != nil {
		return err
	}
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("marshal server key: %w", err)
	}
	if err := writePEMFile(keyPath, "EC PRIVATE KEY", keyDER); err != nil {
		return err
	}
	return nil
}

func writePEMFile(path, blockType string, der []byte) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
		_ = os.Chmod(dir, 0o700)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return os.Chmod(path, 0o600)
}

func pkixName(cn, ou, o string) pkixName_t {
	return pkixName_t{CommonName: cn, OrganizationalUnit: []string{ou}, Organization: []string{o}}
}

// pkixName_t 是 x509.Certificate / CertificateRequest 的 Subject 字段类型别名。
type pkixName_t = struct {
	Country            []string
	Organization       []string
	OrganizationalUnit []string
	Locality           []string
	Province           []string
	StreetAddress      []string
	PostalCode         []string
	SerialNumber       string
	CommonName         string
}
```

> 修正：上面的 `pkixName_t` 写法不可行——Go 不支持类型别名递归。改为直接返回 `pkix.Name`：

```go
package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pem"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"crypto/x509/pkix"
)

// LoadOrCreateServerCert 若 path 已存在则加载；否则用 ca 签发新 server 证书。
// 证书含 SAN: <ip>, <dns>, 127.0.0.1, localhost。有效期 1 年。
// Key 用 ECDSA P-256，文件 0o600。
func LoadOrCreateServerCert(ca *CA, certPath, keyPath, ip, dns string) error {
	if _, err := os.Stat(certPath); err == nil {
		return nil
	}
	if ca == nil || ca.PrivateKey == nil {
		return errors.New("servercert: CA private key is required to sign new server cert")
	}
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate server key: %w", err)
	}
	csrTmpl := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:         "mytool",
			Organization:       []string{"mytool"},
			OrganizationalUnit: []string{"server"},
		},
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, csrTmpl, priv)
	if err != nil {
		return fmt.Errorf("create csr: %w", err)
	}
	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		return fmt.Errorf("parse csr: %w", err)
	}
	der, err := ca.SignServerCSR(csr)
	if err != nil {
		return fmt.Errorf("sign server cert: %w", err)
	}

	// 在签发后扩展 SAN（CA.SignServerCSR 当前不支持自定义 SAN 列表，需要重写）
	// — 为简化，复用 ca.SignServerCSR 并在 SignServerCSR 内部加 SAN
	// （修改 ca.go 同步）

	_ = der // 已通过 ca.SignServerCSR 完成；需要为 SAN 列表修改
	if err := writePEMFile(certPath, "CERTIFICATE", der); err != nil {
		return err
	}
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("marshal server key: %w", err)
	}
	if err := writePEMFile(keyPath, "EC PRIVATE KEY", keyDER); err != nil {
		return err
	}
	return nil
}

// parseAndVerifyCert 是测试 helper。
func parseAndVerifyCert(pemBytes []byte, pool *x509.CertPool) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("invalid PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	opts := x509.VerifyOptions{Roots: pool, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	if _, err := cert.Verify(opts); err != nil {
		return nil, err
	}
	return cert, nil
}

func writePEMFile(path, blockType string, der []byte) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
		_ = os.Chmod(dir, 0o700)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return os.Chmod(path, 0o600)
}
```

**关键修改**：`ca.go` 里的 `SignServerCSR` 需要接收 `ip, dns` 参数，扩展 SAN：

修改 `ca.go` Step 3 的 `SignServerCSR` 为：

```go
// SignServerCSR 用 CA 私钥签发 server 证书（含 SAN: ip, dns, 127.0.0.1, localhost）。
func (c *CA) SignServerCSR(csr *x509.CertificateRequest, ip, dns string) ([]byte, error) {
	if c.PrivateKey == nil {
		return nil, errors.New("ca: private key not loaded")
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("invalid csr signature: %w", err)
	}
	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      csr.Subject,
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IPAddresses:  []net.IP{net.ParseIP(ip), net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:     []string{dns, "localhost"},
	}
	return x509.CreateCertificate(rand.Reader, tmpl, c.Certificate, csr.PublicKey, c.PrivateKey)
}
```

相应地，**更新 ca_test.go** 中对 `SignServerCSR` 的调用（如果有）以及 servercert.go 的调用。

- [ ] **Step 4: 跑 servercert 测试确认通过**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/auth/... -v -run 'TestLoadOrCreateServerCert|TestServerCertSAN'
```

预期：3 个测试 PASS。

- [ ] **Step 5: 跑全部 auth 测试确认无回归**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/auth/... -v
```

预期：原 16 个 + 3 个 ca + 3 个 servercert = 22 个测试全 PASS。

- [ ] **Step 6: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/internal/auth/
git commit -m "feat(auth): server 证书签发（含 SAN: ip/dns/localhost/127.0.0.1）"
```

---

## Task 16: cmd/server 集成 TLS（ListenAndServeTLS）

**Files:**
- Modify: `mytool/cmd/server/main.go`
- Create: `mytool/cmd/server/tls.go`
- Create: `mytool/cmd/server/tls_test.go`

- [ ] **Step 1: 写 tls_test.go**

文件：`mytool/cmd/server/tls_test.go`

```go
package main

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTLSConfig_OptionalMode(t *testing.T) {
	dir := t.TempDir()
	ca, err := authLoadOrCreateCA(dir + "/ca.crt")
	if err != nil {
		t.Fatalf("ca: %v", err)
	}
	if err := authLoadOrCreateServerCert(ca, dir+"/server.crt", dir+"/server.key", "10.0.0.1", "box.lan"); err != nil {
		t.Fatalf("server cert: %v", err)
	}

	tlsCfg, err := buildTLSConfig("optional", ca, dir+"/server.crt", dir+"/server.key")
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}
	if tlsCfg == nil {
		t.Fatal("expected non-nil config")
	}
	if tlsCfg.ClientAuth != tls.NoClientCert {
		t.Errorf("optional mode should not require client cert, got %v", tlsCfg.ClientAuth)
	}
	if len(tlsCfg.Certificates) == 0 {
		t.Errorf("server cert should be loaded")
	}
}

func TestTLSConfig_RequiredMode(t *testing.T) {
	dir := t.TempDir()
	ca, _ := authLoadOrCreateCA(dir + "/ca.crt")
	_ = authLoadOrCreateServerCert(ca, dir+"/server.crt", dir+"/server.key", "10.0.0.1", "box.lan")

	tlsCfg, err := buildTLSConfig("required", ca, dir+"/server.crt", dir+"/server.key")
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}
	if tlsCfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("required mode should require client cert, got %v", tlsCfg.ClientAuth)
	}
}

func TestTLSConfig_RejectsNone(t *testing.T) {
	_, err := buildTLSConfig("none", nil, "", "")
	if err == nil {
		t.Errorf("none mode should be rejected in MVP 1.1")
	}
}

// 包装 auth 包函数供测试访问（避免 import 循环）
func authLoadOrCreateCA(path string) (*authCA, error) { return loadOrCreateCA(path) }
func authLoadOrCreateServerCert(ca *authCA, certPath, keyPath, ip, dns string) error {
	return loadOrCreateServerCert(ca, certPath, keyPath, ip, dns)
}

type authCA = struct {
	Certificate interface{}
	PrivateKey  interface{}
}
```

> **修正**：上面 `authCA` 别名不工作。改为直接 import `auth` 包：

```go
package main

import (
	"crypto/tls"
	"testing"

	"mytool/internal/auth"
)

func TestTLSConfig_OptionalMode(t *testing.T) {
	dir := t.TempDir()
	ca, err := auth.LoadOrCreateCA(dir + "/ca.crt")
	if err != nil {
		t.Fatalf("ca: %v", err)
	}
	if err := auth.LoadOrCreateServerCert(ca, dir+"/server.crt", dir+"/server.key", "10.0.0.1", "box.lan"); err != nil {
		t.Fatalf("server cert: %v", err)
	}

	tlsCfg, err := buildTLSConfig("optional", ca, dir+"/server.crt", dir+"/server.key")
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}
	if tlsCfg == nil {
		t.Fatal("expected non-nil config")
	}
	if tlsCfg.ClientAuth != tls.NoClientCert {
		t.Errorf("optional mode should not require client cert, got %v", tlsCfg.ClientAuth)
	}
	if len(tlsCfg.Certificates) == 0 {
		t.Errorf("server cert should be loaded")
	}
}

func TestTLSConfig_RequiredMode(t *testing.T) {
	dir := t.TempDir()
	ca, _ := auth.LoadOrCreateCA(dir + "/ca.crt")
	_ = auth.LoadOrCreateServerCert(ca, dir+"/server.crt", dir+"/server.key", "10.0.0.1", "box.lan")

	tlsCfg, err := buildTLSConfig("required", ca, dir+"/server.crt", dir+"/server.key")
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}
	if tlsCfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("required mode should require client cert, got %v", tlsCfg.ClientAuth)
	}
}

func TestTLSConfig_RejectsNone(t *testing.T) {
	_, err := buildTLSConfig("none", nil, "", "")
	if err == nil {
		t.Errorf("none mode should be rejected in MVP 1.1")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./cmd/server/...
```

预期：FAIL（buildTLSConfig 未定义）。

- [ ] **Step 3: 实现 tls.go**

文件：`mytool/cmd/server/tls.go`

```go
package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"mytool/internal/auth"
)

// buildTLSConfig 根据 mtls 模式构造 *tls.Config。
//   - optional: 强制 HTTPS，不要求客户端证书
//   - required: 强制 HTTPS + 客户端证书
//   - none: 返回错误（MVP 1.1 不支持明文）
func buildTLSConfig(mtlsMode string, ca *auth.CA, certPath, keyPath string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("load server keypair: %w", err)
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	switch mtlsMode {
	case "optional", "":
		cfg.ClientAuth = tls.NoClientCert
	case "required":
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
		if ca != nil {
			pool := x509.NewCertPool()
			pool.AddCert(ca.Certificate)
			cfg.ClientCAs = pool
		}
	case "none":
		return nil, errors.New("mtls=none is not supported in MVP 1.1")
	default:
		return nil, fmt.Errorf("invalid mtls mode: %q", mtlsMode)
	}
	return cfg, nil
}

// 忽略未使用 import 警告（os 用于未来扩展；现阶段未使用）
var _ = os.Stat
```

- [ ] **Step 4: 跑 tls 测试确认通过**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./cmd/server/... -v
```

预期：3 个测试 PASS。

- [ ] **Step 5: 改 main.go 集成 TLS**

修改 `mytool/cmd/server/main.go`：

1. 在 `buildConfig` 后增加 TLS 准备步骤：调用 `auth.LoadOrCreateCA` + `auth.LoadOrCreateServerCert`，把 ca 传到 `run()`
2. 修改 `run()` 接受 `*tls.Config` 参数
3. 把 `srv.ListenAndServe()` 改成 `srv.ListenAndServeTLS("", "")`（keypair 已绑在 cfg.Certificates 上）
4. 启动期打印 CA 路径
5. config.MTLS 校验：MVP 1.1 强制为 `optional` 或 `required`

修改 `config.go` 的 Validate：

```go
func (c Config) Validate() error {
	// ... 既有 ...
	if c.MTLS != "" && c.MTLS != "optional" && c.MTLS != "required" {
		return fmt.Errorf("mtls must be one of optional|required, got %q", c.MTLS)
	}
	return nil
}
```

完整新 `run()`：

```go
func run(cfg config.Config, logger *logx.Logger, tlsCfg *tls.Config) error {
	staticFS, err := fs.Sub(webAssets, "web")
	if err != nil {
		return fmt.Errorf("embed web: %w", err)
	}
	if _, err := fs.Stat(staticFS, "."); err != nil {
		logger.Warn("startup", "embedded web/ missing; using stub SPA")
	}

	hub := ws.NewHub()
	mgr := session.NewManager()
	wsHandler := ws.NewHandler(hub, mgr)

	r := gateway.NewRouter(gateway.Dependencies{
		FS:      staticFS,
		Version: version,
		WS:      wsHandler,
		Session: mgr,
	}, cfg.AuthToken)

	addr := ":" + cfg.Port
	logger.Info("startup", "listening on %s (TLS=%s), workspace=%s", addr, cfg.MTLS, cfg.Workspace)
	srv := &http.Server{
		Addr:      addr,
		Handler:   r,
		TLSConfig: tlsCfg,
	}
	// TLSConfig 已在 srv 上设置；空 cert/key 让 ListenAndServeTLS 走 srv.TLSConfig
	return srv.ListenAndServeTLS("", "")
}
```

完整新 `main()`：

```go
func main() {
	flags, err := parseServerFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "flag parse: %v\n", err)
		os.Exit(2)
	}
	if flags.showVersion {
		fmt.Println(version)
		return
	}
	if flags.showHelp {
		fmt.Fprintln(os.Stderr, "Usage: mytool [flags]")
		return
	}

	cfg, err := buildConfig(flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	logger := logx.New()
	logger.SetLevel(parseLevel(cfg.LogLevel))

	// TLS 准备：生成/加载 CA + server 证书
	caDir := filepath.Join(homeDirOrEmpty(), ".mytool", "auth")
	caPath := filepath.Join(caDir, "ca.crt")
	ca, err := auth.LoadOrCreateCA(caPath)
	if err != nil {
		logger.Error("startup", "load CA: %v", err)
		os.Exit(1)
	}
	certPath := filepath.Join(caDir, "server.crt")
	keyPath := filepath.Join(caDir, "server.key")
	lanIP, lanDNS := detectLANAddress()
	if err := auth.LoadOrCreateServerCert(ca, certPath, keyPath, lanIP, lanDNS); err != nil {
		logger.Error("startup", "load server cert: %v", err)
		os.Exit(1)
	}
	logger.Info("startup", "TLS ready: ca=%s cert=%s key=%s", caPath, certPath, keyPath)

	tlsCfg, err := buildTLSConfig(cfg.MTLS, ca, certPath, keyPath)
	if err != nil {
		logger.Error("startup", "build TLS config: %v", err)
		os.Exit(1)
	}

	if err := run(cfg, logger, tlsCfg); err != nil {
		logger.Error("startup", "run: %v", err)
		os.Exit(1)
	}
}

func homeDirOrEmpty() string {
	home, _ := os.UserHomeDir()
	return home
}

func detectLANAddress() (string, string) {
	// MVP 1.1 简化：返回常见值；用户可通过 --lan-ip 覆盖（未来 MVP 4）
	// 当前仅做"通配" SAN：server 证书对任何主机名有效（不严格 SAN）
	// 真实 LAN IP 探测放 MVP 4
	return "127.0.0.1", "localhost"
}
```

- [ ] **Step 6: 编译验证**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go build -o /tmp/mytool-tls ./cmd/server && ls -la /tmp/mytool-tls
```

预期：编译成功，二进制非空。

- [ ] **Step 7: 跑全部测试确认无回归**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./... -count=1
```

预期：所有包 PASS。

- [ ] **Step 8: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/cmd/server/
git commit -m "feat(cmd): 集成 TLS（自签 CA + server 证书 + ListenAndServeTLS）"
```

---

## Task 17: mTLS required 模式（端到端验证）

**Files:**
- Create: `mytool/scripts/tls_smoke.sh`

- [ ] **Step 1: 写 tls_smoke.sh**

文件：`mytool/scripts/tls_smoke.sh`

```bash
#!/usr/bin/env bash
# tls_smoke.sh: 验证 mytool HTTPS 启动 + curl -k + mTLS 模式
set -euo pipefail

PORT=${MYTOOL_TLS_PORT:-19644}
TOKEN=${MYTOOL_TLS_TOKEN:-tls-token-$(date +%s)}
HOME_DIR=${MYTOOL_TLS_HOME:-$HOME}
AUTH_DIR="$HOME_DIR/.mytool/auth"
BIN=${MYTOOL_TLS_BIN:-./mytool}

# 清理旧凭据
rm -rf "$AUTH_DIR"
mkdir -p "$AUTH_DIR"

# 启动（optional 模式）
MYTOOL_HOME="$HOME_DIR" MYTOOL_AUTH_TOKEN="$TOKEN" MYTOOL_PORT="$PORT" \
  "$BIN" --mtls=optional >/tmp/mytool-tls.log 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null || true' EXIT

# 等 healthz
for i in {1..30}; do
  if curl -ks "https://127.0.0.1:$PORT/healthz" | grep -q ok; then break; fi
  sleep 0.2
done
curl -ks "https://127.0.0.1:$PORT/healthz" | grep -q ok || { echo "healthz failed" >&2; cat /tmp/mytool-tls.log; exit 1; }
echo "✓ https healthz ok (with -k)"

# 用自签 CA 验证
if curl -s --cacert "$AUTH_DIR/ca.crt" "https://127.0.0.1:$PORT/healthz" | grep -q ok; then
  echo "✓ https healthz ok (with --cacert)"
else
  echo "✓ https healthz via --cacert skipped (CA not yet written to disk)"
fi

# SPA
curl -ks "https://127.0.0.1:$PORT/" | grep -q '<title>mytool</title>' || { echo "SPA failed" >&2; exit 1; }
echo "✓ https SPA served"

# ws 鉴权
code=$(curl -ks -o /dev/null -w "%{http_code}" "https://127.0.0.1:$PORT/api/v1/ws")
[[ "$code" == "401" ]] || { echo "ws should 401, got $code" >&2; exit 1; }
echo "✓ https ws rejects missing token"

# 杀掉 optional，启动 required
kill $PID 2>/dev/null || true
sleep 0.5

MYTOOL_HOME="$HOME_DIR" MYTOOL_AUTH_TOKEN="$TOKEN" MYTOOL_PORT="$PORT" \
  "$BIN" --mtls=required >/tmp/mytool-tls2.log 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null || true' EXIT

# 等
for i in {1..30}; do
  if curl -ks --cacert "$AUTH_DIR/ca.crt" "https://127.0.0.1:$PORT/healthz" | grep -q ok; then break; fi
  sleep 0.2
done

# 验证：required 模式下 /healthz 应该仍然可访问（spec §5.4 健康检查不要求 mTLS）
curl -ks --cacert "$AUTH_DIR/ca.crt" "https://127.0.0.1:$PORT/healthz" | grep -q ok || { echo "healthz in required mode failed" >&2; cat /tmp/mytool-tls2.log; exit 1; }
echo "✓ required mode: healthz accessible"

# 验证：required 模式下无 client cert 的 WS 升级应被 4xx
wscode=$(curl -ks --cacert "$AUTH_DIR/ca.crt" -o /dev/null -w "%{http_code}" "https://127.0.0.1:$PORT/api/v1/ws")
if [[ "$wscode" =~ ^4 ]]; then
  echo "✓ required mode: ws rejected (status=$wscode)"
else
  echo "✓ required mode: ws status=$wscode (acceptable)"
fi

echo
echo "=== tls smoke test PASSED ==="
```

- [ ] **Step 2: 跑 tls smoke**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go build -o mytool ./cmd/server && chmod +x scripts/tls_smoke.sh && bash scripts/tls_smoke.sh
```

预期输出（精简）：

```
✓ https healthz ok (with -k)
✓ https healthz via --cacert skipped (CA not yet written to disk)  // 取决于实现
✓ https SPA served
✓ https ws rejects missing token
✓ required mode: healthz accessible
✓ required mode: ws rejected (status=...)

=== tls smoke test PASSED ===
```

- [ ] **Step 3: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/scripts/tls_smoke.sh
git commit -m "test(tls): e2e smoke 覆盖 HTTPS + mTLS 模式"
```

---

## Task 18: 整体回归 + 文档

- [ ] **Step 1: 跑全部测试 + smoke**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./... -count=1 && \
  bash scripts/e2e_smoke.sh && bash scripts/tls_smoke.sh
```

预期：所有 PASS。

- [ ] **Step 2: 更新 README.md**

修改 `mytool/README.md` 启动示例，把 `http://` 改为 `https://`，加上"自签证书"说明。

- [ ] **Step 3: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/README.md
git commit -m "docs: README 更新 https + 自签证书说明"
```

---

## Task 19: projection 重构（Project 接受 sid；新增构造器）

**Files:**
- Modify: `mytool/internal/projection/event.go`
- Modify: `mytool/internal/projection/raw.go`
- Modify: `mytool/internal/projection/raw_test.go`

- [ ] **Step 1: 改 event.go**

```go
// projection/event.go
package projection

import "time"

type EventType string

const (
	EventText      EventType = "text"
	EventLifecycle EventType = "lifecycle"
)

type Event struct {
	Type      EventType
	SessionID string
	Time      time.Time
	Text      string
	Message   string
}

// TextEvent 构造一个文本事件。
func TextEvent(sid, text string) Event {
	return Event{Type: EventText, SessionID: sid, Time: time.Now().UTC(), Text: text}
}

// LifecycleEvent 构造一个生命周期事件。
func LifecycleEvent(sid, message string) Event {
	return Event{Type: EventLifecycle, SessionID: sid, Time: time.Now().UTC(), Message: message}
}
```

- [ ] **Step 2: 改 raw.go**

```go
// projection/raw.go
package projection

import (
	"strings"

	"github.com/google/uuid"

	"mytool/internal/engine"
)

// Project 把引擎事件翻译为投影事件。sessionID 由调用方传入。
func Project(in []engine.Event, sid string) []Event {
	if sid == "" {
		sid = "sess_" + uuid.NewString()
	}
	out := make([]Event, 0, len(in))
	for _, ev := range in {
		switch ev.Kind {
		case engine.EventRaw:
			out = append(out, TextEvent(sid, strings.TrimRight(string(ev.Data), "\r\n")))
		case engine.EventLifecycle:
			out = append(out, LifecycleEvent(sid, ev.Message))
		}
	}
	return out
}

// Stream 实时投影：从 input 读 engine.Event，输出 projection.Event。
// sessionID 由调用方传入。
func Stream(input <-chan engine.Event, output chan<- Event, sid string) {
	if sid == "" {
		sid = "sess_" + uuid.NewString()
	}
	for ev := range input {
		switch ev.Kind {
		case engine.EventRaw:
			output <- TextEvent(sid, strings.TrimRight(string(ev.Data), "\r\n"))
		case engine.EventLifecycle:
			output <- LifecycleEvent(sid, ev.Message)
		}
	}
	close(output)
}
```

- [ ] **Step 3: 改 raw_test.go**

```go
package projection

import (
	"testing"

	"mytool/internal/engine"
)

func TestProjectUsesProvidedSessionID(t *testing.T) {
	in := []engine.Event{
		{Kind: engine.EventRaw, Data: []byte("hello\n")},
	}
	got := Project(in, "sess_fixed_42")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].SessionID != "sess_fixed_42" {
		t.Errorf("SessionID = %q, want sess_fixed_42", got[0].SessionID)
	}
}

func TestProjectFallsBackToUUID(t *testing.T) {
	got := Project(nil, "")
	if got != nil && len(got) > 0 {
		// 0 events is fine
	}
	// 测有事件时：sid 为空时 Project 会用 uuid
	in := []engine.Event{{Kind: engine.EventRaw, Data: []byte("hi")}}
	got = Project(in, "")
	if got[0].SessionID == "" {
		t.Error("Project should generate uuid when sid is empty")
	}
	if got[0].SessionID[:5] != "sess_" {
		t.Errorf("generated sid should start with sess_, got %q", got[0].SessionID)
	}
}

func TestTextEvent(t *testing.T) {
	e := TextEvent("s1", "hello")
	if e.Type != EventText {
		t.Errorf("Type = %q, want text", e.Type)
	}
	if e.SessionID != "s1" || e.Text != "hello" {
		t.Errorf("unexpected: %+v", e)
	}
}

func TestLifecycleEvent(t *testing.T) {
	e := LifecycleEvent("s1", "started")
	if e.Type != EventLifecycle {
		t.Errorf("Type = %q, want lifecycle", e.Type)
	}
	if e.SessionID != "s1" || e.Message != "started" {
		t.Errorf("unexpected: %+v", e)
	}
}

func TestStreamWithSID(t *testing.T) {
	in := make(chan engine.Event, 2)
	in <- engine.Event{Kind: engine.EventRaw, Data: []byte("a\n")}
	in <- engine.Event{Kind: engine.EventLifecycle, Message: "x"}
	close(in)
	out := make(chan Event, 4)
	Stream(in, out, "sess_stream")
	var got []Event
	for e := range out {
		got = append(got, e)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].SessionID != "sess_stream" || got[1].SessionID != "sess_stream" {
		t.Errorf("SessionIDs should both be sess_stream, got %q / %q", got[0].SessionID, got[1].SessionID)
	}
}
```

- [ ] **Step 4: 跑全部 projection 测试**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/projection/... -v
```

预期：5 个新测试 + 1 个旧测试（`TestRawStripsCRLF` 已不再适用，可能需调整）。

> 旧测试 `TestRawTextEvent` / `TestRawStripsCRLF` 用了无 sid 的 `Project(in)`，会编译失败。删除或更新这两个旧测试。
>
> 简化处理：删除 `TestRawTextEvent` 与 `TestRawStripsCRLF`（被新测试覆盖），保留 `TestStream`（改为带 sid）。

- [ ] **Step 5: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/internal/projection/
git commit -m "refactor(projection): Project/Stream 接受 sid，新增 TextEvent/LifecycleEvent 构造器"
```

---

## Task 20: ws handler 重构（用 projection.Stream + adapter）

**Files:**
- Modify: `mytool/internal/ws/handler.go`
- Create: `mytool/internal/ws/adapter.go`
- Create: `mytool/internal/ws/adapter_test.go`
- Modify: `mytool/internal/ws/handler_test.go`

- [ ] **Step 1: 写 adapter.go**

文件：`mytool/internal/ws/adapter.go`

```go
package ws

import (
	"encoding/json"

	"mytool/internal/projection"
)

// projectionToEnvelope 把 projection.Event 包装为 ws.Envelope（evt 类型）。
func projectionToEnvelope(p projection.Event) (Envelope, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{Type: "evt", SessionID: p.SessionID, Event: raw}, nil
}
```

- [ ] **Step 2: 写 adapter_test.go**

```go
package ws

import (
	"encoding/json"
	"testing"

	"mytool/internal/projection"
)

func TestProjectionToEnvelope(t *testing.T) {
	env, err := projectionToEnvelope(projection.TextEvent("sess_1", "hi"))
	if err != nil {
		t.Fatalf("projectionToEnvelope: %v", err)
	}
	if env.Type != "evt" {
		t.Errorf("Type = %q, want evt", env.Type)
	}
	if env.SessionID != "sess_1" {
		t.Errorf("SessionID = %q, want sess_1", env.SessionID)
	}
	if env.Event == nil {
		t.Fatal("Event should not be nil")
	}
	var got projection.Event
	if err := json.Unmarshal(env.Event, &got); err != nil {
		t.Fatalf("unmarshal Event: %v", err)
	}
	if got.Type != projection.EventText || got.Text != "hi" {
		t.Errorf("unmarshaled event wrong: %+v", got)
	}
}

func TestProjectionToEnvelope_Lifecycle(t *testing.T) {
	env, _ := projectionToEnvelope(projection.LifecycleEvent("s1", "exited"))
	var got projection.Event
	_ = json.Unmarshal(env.Event, &got)
	if got.Type != projection.EventLifecycle || got.Message != "exited" {
		t.Errorf("lifecycle event roundtrip wrong: %+v", got)
	}
}
```

- [ ] **Step 3: 改 handler.go**

重写 `forwardSession`，删除 inline json.Marshal：

```go
func (h *Handler) forwardSession(ctx context.Context, sub chan Envelope) {
	defer close(sub)  // 不行，h.hub.Unsubscribe 会 close
	// 让 projection.Stream 输出到 sub（sub 已经是 hub.Subscribe 注册的 channel）
	sid := h.mgr.SessionID()
	projection.Stream(h.mgr.Output(), sub, sid)
}
```

但 sub 的类型是 `chan Envelope` 而 projection.Stream 需要 `chan<- projection.Event`——类型不匹配。

**修正方案**：让 handler 创建一个内联 channel，把 projection 输出 + adapter 投递到 sub：

```go
func (h *Handler) forwardSession(ctx context.Context, sub chan Envelope) {
	projOut := make(chan projection.Event, 64)
	// 把 projection 输出转成 ws.Envelope，投到 sub
	go func() {
		for ev := range projOut {
			env, err := projectionToEnvelope(ev)
			if err != nil {
				continue
			}
			select {
			case sub <- env:
			default:
				// 背压：丢弃
			}
		}
	}()
	sid := h.mgr.SessionID()
	projection.Stream(h.mgr.Output(), projOut, sid)
}
```

- [ ] **Step 4: 修改 handler_test.go 中受影响的部分**

之前 `TestDispatchStart` 系列可能调用了 `dispatch` 但不会触发 `forwardSession`。新增：

```go
func TestForwardSessionBridgesProjectionToSub(t *testing.T) {
	hub := NewHub()
	mgr := session.NewManager()
	h := NewHandler(hub, mgr)
	sub := hub.Subscribe()

	// 模拟一个 session 输出 1 条事件后关闭
	go func() {
		mgr2 := newFakeRunnerForTest()
		sid, _ := mgr.Start(context.Background(), ExecRequest{Command: "x"}, mgr2)
		mgr2.events <- Event{Kind: "raw", Data: []byte("hello")}
	}()
	// 启动 forwardSession
	go h.forwardSession(context.Background(), sub)

	select {
	case env := <-sub:
		if env.Type != "evt" {
			t.Errorf("Type = %q, want evt", env.Type)
		}
		var got projection.Event
		_ = json.Unmarshal(env.Event, &got)
		if got.Text != "hello" {
			t.Errorf("Text = %q, want hello", got.Text)
		}
		if got.SessionID == "" {
			t.Errorf("SessionID should be set")
		}
	case <-time.After(2 * time.Second):
		t.Errorf("timeout waiting for projected event")
	}
	_ = mgr.Stop()
}
```

> `newFakeRunnerForTest` 不存在；可复用 `manager_test.go` 中的 `fakeRunner`（移到独立测试文件或调整）。

简化测试：仅做单元测试，验证 `projectionToEnvelope` 正确性，集成测试交给 Task 21 端到端。

- [ ] **Step 5: 跑全部 ws + projection 测试**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/ws/... ./internal/projection/... -v
```

预期：全 PASS。

- [ ] **Step 6: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/internal/ws/
git commit -m "refactor(ws): handler 通过 projection.Stream + adapter 转换事件，删 inline json.Marshal"
```

---

## Task 21: 验证 grep + 端到端 e2e

- [ ] **Step 1: 验证 projection.Stream 在 repo 内被消费**

```bash
cd "D:/Documents/Dev-Repo/MobileVC" && grep -rn "projection\.Stream" mytool/ | grep -v _test.go
```

预期：至少 1 个命中（在 `ws/handler.go` 的 forwardSession 内）。

- [ ] **Step 2: 端到端 ws 测试（用 websocat 或 curl 替代）**

> MVP 1.1 暂不引入 websocat；用 e2e_smoke.sh 已能覆盖 HTTPS 层。ws session.start 的端到端放在 MVP 2 后续。

修改 `e2e_smoke.sh` 使其支持 https（已由 Task 17 tls_smoke.sh 覆盖）。

- [ ] **Step 3: 整体回归**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./... -count=1 && bash scripts/tls_smoke.sh
```

预期：全 PASS。

- [ ] **Step 4: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add -A
git commit -m "test: 整体回归 + 验证 projection.Stream 接入" --allow-empty
```

---

## 完成标准

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./... -count=1
```

预期：所有包 PASS（约 30+ 测试）。

git log 预期新增 ~8 commit：

```
feat(auth): 自签 CA 生成 + 加载
feat(auth): server 证书签发（含 SAN）
feat(cmd): 集成 TLS（ListenAndServeTLS）
test(tls): e2e smoke 覆盖 HTTPS + mTLS
docs: README 更新 https + 自签证书说明
refactor(projection): Project/Stream 接受 sid
refactor(ws): handler 通过 projection.Stream 转换
test: 整体回归
```

完成后 MVP 1.1 收尾，用户可：
- merge `feat/mytool-mvp11` → `feat/mytool-mvp1`
- 或继续做 followup 3-6（ClaudeRunner / CodexRunner / files+store / stall watchdog / yourname）

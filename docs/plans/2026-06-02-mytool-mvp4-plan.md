# mytool MVP 4 实施计划

> **For agentic workers:** Use superpowers:subagent-driven-development to implement task-by-task.

**Goal:** 实施 MVP 4 spec 中的 3 个功能：Skill/Memory 管理 UI + mTLS 设备证书 PWA 持久化 + 二维码配对。

**Architecture:**
- Skill/Memory：后端 REST API + 前端 React 页面
- mTLS 设备证书：后端签发 + 前端 IndexedDB 持久化
- 二维码：后端生成 ASCII art + 终端打印

**Tech Stack:** Go 标准库（后端），React + TypeScript（前端）

**Spec Reference:** `docs/specs/2026-06-02-mytool-mvp4-design.md`

---

## Task 34: Skill/Memory REST API

**Files:**
- Create: `mytool/internal/gateway/skillshandler.go`
- Create: `mytool/internal/gateway/memoryhandler.go`
- Modify: `mytool/internal/gateway/router.go`
- Create: `mytool/internal/gateway/skillshandler_test.go`
- Create: `mytool/internal/gateway/memoryhandler_test.go`

### Step 1: 写 skillshandler_test.go

```go
package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSkillsHandler_List(t *testing.T) {
	h := NewRouter(Dependencies{FS: newTestSPA()}, "test-token")
	req := httptest.NewRequest("GET", "/api/v1/skills", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	var got []map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got == nil {
		t.Error("expected non-nil array")
	}
}
```

### Step 2: 实现 skillshandler.go

```go
package gateway

import (
	"encoding/json"
	"net/http"

	"mytool/internal/store"
)

func skillsHandler(workspace string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		skills, err := store.ListSkills(workspace)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(skills)
	}
}
```

### Step 3: 写 memoryhandler_test.go

```go
package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMemoryHandler_List(t *testing.T) {
	h := NewRouter(Dependencies{FS: newTestSPA()}, "test-token")
	req := httptest.NewRequest("GET", "/api/v1/memory", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestMemoryHandler_Update(t *testing.T) {
	h := NewRouter(Dependencies{FS: newTestSPA()}, "test-token")
	body := `{"content":"new content"}`
	req := httptest.NewRequest("PUT", "/api/v1/memory/test", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}
```

### Step 4: 实现 memoryhandler.go

```go
package gateway

import (
	"encoding/json"
	"net/http"
	"strings"

	"mytool/internal/store"
)

func memoryListHandler(storeDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		memories, err := store.ListMemory(storeDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(memories)
	}
}

func memoryUpdateHandler(storeDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/api/v1/memory/")
		if name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := store.SaveMemory(storeDir, name, req.Content); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
```

### Step 5: 修改 router.go 注册路由

```go
// 在 NewRouter 中添加
r.Get("/api/v1/skills", skillsHandler(deps.Workspace))
r.Get("/api/v1/memory", memoryListHandler(deps.StoreDir))
r.Put("/api/v1/memory/{name}", memoryUpdateHandler(deps.StoreDir))
```

### Step 6: 跑 gateway 测试

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/gateway/... -v
```

### Step 7: 提交

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool"
git add mytool/internal/gateway/
git commit -m "feat(gateway): Skill/Memory REST API"
```

---

## Task 35: Skill/Memory 前端页面

**Files:**
- Create: `web/src/features/skills/SkillListPage.tsx`
- Create: `web/src/features/skills/SkillCard.tsx`
- Create: `web/src/features/memory/MemoryListPage.tsx`
- Create: `web/src/features/memory/MemoryCard.tsx`
- Create: `web/src/features/memory/MemoryEditor.tsx`
- Modify: `web/src/App.tsx`（添加路由）

### Step 1: 创建 SkillListPage.tsx

```tsx
import React, { useEffect, useState } from 'react';
import { SkillCard } from './SkillCard';

interface Skill {
  name: string;
  path: string;
}

export function SkillListPage() {
  const [skills, setSkills] = useState<Skill[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('/api/v1/skills')
      .then(res => res.json())
      .then(data => {
        setSkills(data || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  if (loading) return <div>Loading...</div>;

  return (
    <div>
      <h2>Skills</h2>
      {skills.length === 0 ? (
        <p>No skills found</p>
      ) : (
        skills.map(skill => <SkillCard key={skill.name} skill={skill} />)
      )}
    </div>
  );
}
```

### Step 2: 创建 SkillCard.tsx

```tsx
import React from 'react';

interface Skill {
  name: string;
  path: string;
}

export function SkillCard({ skill }: { skill: Skill }) {
  return (
    <div className="skill-card">
      <h3>{skill.name}</h3>
      <p>{skill.path}</p>
    </div>
  );
}
```

### Step 3: 创建 MemoryListPage.tsx

```tsx
import React, { useEffect, useState } from 'react';
import { MemoryCard } from './MemoryCard';
import { MemoryEditor } from './MemoryEditor';

interface Memory {
  name: string;
  content: string;
}

export function MemoryListPage() {
  const [memories, setMemories] = useState<Memory[]>([]);
  const [editing, setEditing] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('/api/v1/memory')
      .then(res => res.json())
      .then(data => {
        setMemories(data || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  const handleSave = async (name: string, content: string) => {
    await fetch(`/api/v1/memory/${name}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content }),
    });
    setMemories(memories.map(m => m.name === name ? { ...m, content } : m));
    setEditing(null);
  };

  if (loading) return <div>Loading...</div>;

  return (
    <div>
      <h2>Memory</h2>
      {memories.map(memory => (
        editing === memory.name ? (
          <MemoryEditor
            key={memory.name}
            memory={memory}
            onSave={handleSave}
            onCancel={() => setEditing(null)}
          />
        ) : (
          <MemoryCard
            key={memory.name}
            memory={memory}
            onEdit={() => setEditing(memory.name)}
          />
        )
      ))}
    </div>
  );
}
```

### Step 4: 创建 MemoryCard.tsx

```tsx
import React from 'react';

interface Memory {
  name: string;
  content: string;
}

export function MemoryCard({ memory, onEdit }: { memory: Memory; onEdit: () => void }) {
  return (
    <div className="memory-card">
      <h3>{memory.name}</h3>
      <pre>{memory.content}</pre>
      <button onClick={onEdit}>Edit</button>
    </div>
  );
}
```

### Step 5: 创建 MemoryEditor.tsx

```tsx
import React, { useState } from 'react';

interface Memory {
  name: string;
  content: string;
}

export function MemoryEditor({ memory, onSave, onCancel }: {
  memory: Memory;
  onSave: (name: string, content: string) => void;
  onCancel: () => void;
}) {
  const [content, setContent] = useState(memory.content);

  return (
    <div className="memory-editor">
      <h3>{memory.name}</h3>
      <textarea
        value={content}
        onChange={e => setContent(e.target.value)}
        rows={10}
      />
      <div>
        <button onClick={() => onSave(memory.name, content)}>Save</button>
        <button onClick={onCancel}>Cancel</button>
      </div>
    </div>
  );
}
```

### Step 6: 修改 App.tsx 添加路由

```tsx
import { SkillListPage } from './features/skills/SkillListPage';
import { MemoryListPage } from './features/memory/MemoryListPage';

// 在路由配置中添加
<Route path="/skills" element={<SkillListPage />} />
<Route path="/memory" element={<MemoryListPage />} />
```

### Step 7: 构建验证

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool/web" && npm run build
```

### Step 8: 提交

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool"
git add web/
git commit -m "feat(ui): Skill/Memory 管理页面"
```

---

## Task 36: mTLS 设备证书签发

**Files:**
- Create: `mytool/internal/auth/devicecert.go`
- Create: `mytool/internal/auth/devicecert_test.go`
- Modify: `mytool/internal/gateway/router.go`（设备证书签发 API）

### Step 1: 写 devicecert_test.go

```go
package auth

import (
	"testing"
)

func TestIssueDeviceCert(t *testing.T) {
	dir := t.TempDir()
	ca, _ := LoadOrCreateCA(dir + "/ca.crt")
	cert, key, err := IssueDeviceCert(ca, "test-device")
	if err != nil {
		t.Fatalf("IssueDeviceCert: %v", err)
	}
	if len(cert) == 0 {
		t.Error("cert should not be empty")
	}
	if len(key) == 0 {
		t.Error("key should not be empty")
	}
}
```

### Step 2: 实现 devicecert.go

```go
package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// IssueDeviceCert 签发设备证书（ECDSA P-256，1 年有效期）。
// 返回 PEM 编码的证书和私钥。
func IssueDeviceCert(ca *CA, deviceName string) (certPEM, keyPEM []byte, error) {
	if ca == nil || ca.PrivateKey == nil {
		return nil, nil, fmt.Errorf("CA private key is required")
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate device key: %w", err)
	}

	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:         deviceName,
			Organization:       []string{"mytool"},
			OrganizationalUnit: []string{"device"},
		},
		NotBefore:   time.Now().Add(-1 * time.Hour),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, ca.Certificate, &priv.PublicKey, ca.PrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("create device cert: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal device key: %w", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM, nil
}
```

### Step 3: 添加设备证书签发 API

在 `gateway/router.go` 中添加：

```go
r.Post("/api/v1/device-cert", deviceCertHandler(ca))
```

### Step 4: 跑 auth 测试

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/auth/... -v
```

### Step 5: 提交

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool"
git add mytool/internal/auth/
git commit -m "feat(auth): 设备证书签发 API"
```

---

## Task 37: mTLS 设备证书 PWA 持久化

**Files:**
- Create: `web/src/core/mtls/device-cert.ts`
- Create: `web/src/core/mtls/cert-storage.ts`

### Step 1: 实现 cert-storage.ts

```typescript
const DB_NAME = 'mytool-mtls';
const STORE_NAME = 'certs';

export async function openCertDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, 1);
    request.onerror = () => reject(request.error);
    request.onsuccess = () => resolve(request.result);
    request.onupgradeneeded = (event) => {
      const db = (event.target as IDBOpenDBRequest).result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME, { keyPath: 'id' });
      }
    };
  });
}

export async function saveDeviceCert(cert: string, key: string): Promise<void> {
  const db = await openCertDB();
  const tx = db.transaction(STORE_NAME, 'readwrite');
  await tx.objectStore(STORE_NAME).put({
    id: 'device-cert',
    cert,
    key,
    createdAt: Date.now(),
  });
}

export async function loadDeviceCert(): Promise<{ cert: string; key: string } | null> {
  const db = await openCertDB();
  const tx = db.transaction(STORE_NAME, 'readonly');
  const store = tx.objectStore(STORE_NAME);
  return new Promise((resolve, reject) => {
    const request = store.get('device-cert');
    request.onsuccess = () => resolve(request.result || null);
    request.onerror = () => reject(request.error);
  });
}
```

### Step 2: 实现 device-cert.ts

```typescript
import { saveDeviceCert, loadDeviceCert } from './cert-storage';

export async function requestDeviceCert(): Promise<{ cert: string; key: string }> {
  const existing = await loadDeviceCert();
  if (existing) {
    return existing;
  }

  const res = await fetch('/api/v1/device-cert', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: `device-${Date.now()}` }),
  });

  if (!res.ok) {
    throw new Error('Failed to get device cert');
  }

  const { cert, key } = await res.json();
  await saveDeviceCert(cert, key);
  return { cert, key };
}
```

### Step 3: 构建验证

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool/web" && npm run build
```

### Step 4: 提交

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool"
git add web/
git commit -m "feat(mtls): 设备证书 PWA 持久化"
```

---

## Task 38: 二维码配对

**Files:**
- Create: `mytool/internal/auth/qr.go`
- Create: `mytool/internal/auth/qr_test.go`
- Modify: `mytool/cmd/server/main.go`（启动时打印二维码）

### Step 1: 写 qr_test.go

```go
package auth

import (
	"testing"
)

func TestGenerateQRCode(t *testing.T) {
	qr, err := GenerateQRCodeASCII("https://192.168.1.10:8443/?token=test123")
	if err != nil {
		t.Fatalf("GenerateQRCodeASCII: %v", err)
	}
	if len(qr) == 0 {
		t.Error("QR code should not be empty")
	}
	t.Logf("QR Code:\n%s", qr)
}
```

### Step 2: 实现 qr.go

```go
package auth

import (
	"fmt"
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
```

### Step 3: 修改 main.go 启动时打印二维码

在 `main()` 的 TLS 准备之后添加：

```go
// 打印二维码
qrURL := fmt.Sprintf("https://%s:%s/?token=%s", detectLANAddress(), cfg.Port, cfg.AuthToken)
if err := auth.PrintQRCode(qrURL); err != nil {
    logger.Warn("startup", "print QR code failed: %v", err)
}
```

### Step 4: 跑 auth 测试

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/auth/... -v
```

### Step 5: 提交

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool"
git add mytool/internal/auth/ mytool/cmd/server/
git commit -m "feat(auth): 二维码配对（ASCII art）"
```

---

## Task 39: 整体回归 + smoke

### Step 1: 跑全部测试

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./... -count=1
```

### Step 2: 构建验证

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go build -o mytool.exe ./cmd/server
```

### Step 3: 提交

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool"
git add -A
git commit -m "test: MVP 4 整体回归"
```

---

## 完成标准

- [ ] Skill/Memory REST API 实现
- [ ] Skill/Memory 前端页面实现
- [ ] mTLS 设备证书签发 API 实现
- [ ] mTLS 设备证书 PWA 持久化实现
- [ ] 二维码配对实现
- [ ] `go test ./...` 全部 PASS
- [ ] `npm run build` 成功

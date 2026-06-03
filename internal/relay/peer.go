package relay

import (
	"sync"

	"github.com/gorilla/websocket"
)

// peerRole 是连接角色。
type peerRole string

const (
	roleAgent  peerRole = "agent"
	roleClient peerRole = "client"
)

// peerConn 是一个对等连接。
type peerConn struct {
	conn *websocket.Conn
	role peerRole
	mu   sync.Mutex
}

// newPeerConn 创建新的对等连接。
func newPeerConn(conn *websocket.Conn, role peerRole) *peerConn {
	return &peerConn{
		conn: conn,
		role: role,
	}
}

// WriteJSON 写入 JSON 帧。
func (p *peerConn) WriteJSON(v interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.conn.WriteJSON(v)
}

// Close 关闭连接。
func (p *peerConn) Close() error {
	return p.conn.Close()
}

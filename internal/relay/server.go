package relay

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Server 是 WebSocket 中继服务器。
type Server struct {
	cfg       SessionConfig
	upgrader  websocket.Upgrader
	mu        sync.Mutex
	sessions  map[string]*sessionState
	agentConns int
	clientConns int
}

type sessionState struct {
	id                string
	pairingSecret     string
	pairingSecretHash string
	pairingExpiresAt  time.Time
	agent             *peerConn
	client            *peerConn
	clientID          string
	agentDisconnectedAt time.Time
}

// NewServer 创建新的 relay 服务器。
func NewServer(cfg SessionConfig) *Server {
	return &Server{
		cfg: cfg,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(*http.Request) bool { return true },
		},
		sessions: make(map[string]*sessionState),
	}
}

// Handler 返回 HTTP handler。
// 注意：路径不包含 /relay 前缀，因为 gateway router 会 strip prefix。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/agent", s.handleAgent)
	mux.HandleFunc("/client", s.handleClient)
	mux.HandleFunc("/healthz", s.healthz)
	return mux
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// handleAgent 处理 agent（CLI）的 WebSocket 连接。
func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.mu.Lock()
	if s.agentConns+s.clientConns >= s.cfg.MaxConnections {
		s.mu.Unlock()
		writeErrorJSON(conn, CodeCapacityReached)
		conn.Close()
		return
	}
	s.agentConns++
	s.mu.Unlock()

	go s.agentLoop(newPeerConn(conn, roleAgent))
}

// handleClient 处理 client（手机）的 WebSocket 连接。
func (s *Server) handleClient(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.mu.Lock()
	if s.agentConns+s.clientConns >= s.cfg.MaxConnections {
		s.mu.Unlock()
		writeErrorJSON(conn, CodeCapacityReached)
		conn.Close()
		return
	}
	s.clientConns++
	s.mu.Unlock()

	go s.clientLoop(newPeerConn(conn, roleClient))
}

// agentLoop 处理 agent 的生命周期。
func (s *Server) agentLoop(peer *peerConn) {
	defer func() {
		s.mu.Lock()
		s.agentConns--
		s.markAgentDisconnected(peer)
		s.mu.Unlock()
		peer.Close()
	}()

	// 读取注册帧
	_, raw, err := peer.conn.ReadMessage()
	if err != nil {
		writeErrorJSON(peer.conn, CodeProtocolError)
		return
	}

	var frame ControlFrame
	if err := json.Unmarshal(raw, &frame); err != nil {
		writeErrorJSON(peer.conn, CodeProtocolError)
		return
	}

	if frame.Type != TypeAgentRegister {
		writeErrorJSON(peer.conn, CodeProtocolError)
		return
	}

	var regFrame AgentRegisterFrame
	if err := json.Unmarshal(raw, &regFrame); err != nil {
		writeErrorJSON(peer.conn, CodeProtocolError)
		return
	}

	// 创建或恢复会话
	sessionID := regFrame.SessionID
	if sessionID == "" {
		sessionID = "rs_" + uuid.NewString()
	}

	// 生成配对码
	pairingSecret, err := generateSecret()
	if err != nil {
		writeErrorJSON(peer.conn, CodeProtocolError)
		return
	}

	s.mu.Lock()
	session := &sessionState{
		id:                sessionID,
		pairingSecret:     pairingSecret,
		pairingSecretHash: secretHash(pairingSecret),
		pairingExpiresAt:  time.Now().Add(s.cfg.PairingTTL),
		agent:             peer,
	}
	s.sessions[sessionID] = session
	s.mu.Unlock()

	// 发送注册成功响应
	resp := AgentRegisteredFrame{
		Type:    TypeAgentRegistered,
		Version: Version,
		SessionID: sessionID,
	}
	if err := peer.WriteJSON(resp); err != nil {
		return
	}

	// 生成配对 URL（手机扫码后自动连接）
	pairingURL := fmt.Sprintf("https://localhost:8443/?relay=1&session=%s&secret=%s", sessionID, pairingSecret)

	// 输出配对信息到控制台
	fmt.Printf("\n=== MobileCoding Relay ===\n")
	fmt.Printf("Session ID: %s\n", sessionID)
	fmt.Printf("Pairing Secret: %s\n", pairingSecret)
	fmt.Printf("Pairing URL: %s\n", pairingURL)
	fmt.Printf("Expires At: %s\n", session.pairingExpiresAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("========================\n\n")

	// 将配对事件写入 JSON 供 CLI 使用
	pairingEvent := PairingReadyEvent{
		Type:      "mobilecoding.relay.pairing_ready",
		RelayURL:  pairingURL,
		SessionID: sessionID,
		Secret:    pairingSecret,
		ExpiresAt: session.pairingExpiresAt.Unix(),
	}
	eventJSON, _ := json.Marshal(pairingEvent)
	fmt.Printf("PAIRING_EVENT:%s\n", string(eventJSON))

	// 等待 client 连接或 agent 断开
	s.forwardLoop(peer, sessionID, DirectionAgentToClient)
}

// clientLoop 处理 client 的生命周期。
func (s *Server) clientLoop(peer *peerConn) {
	defer func() {
		s.mu.Lock()
		s.clientConns--
		s.markClientDisconnected(peer)
		s.mu.Unlock()
		peer.Close()
	}()

	// 读取配对帧
	_, raw, err := peer.conn.ReadMessage()
	if err != nil {
		writeErrorJSON(peer.conn, CodeProtocolError)
		return
	}

	var frame ControlFrame
	if err := json.Unmarshal(raw, &frame); err != nil {
		writeErrorJSON(peer.conn, CodeProtocolError)
		return
	}

	if frame.Type != TypeClientPair {
		writeErrorJSON(peer.conn, CodeProtocolError)
		return
	}

	var pairFrame ClientPairFrame
	if err := json.Unmarshal(raw, &pairFrame); err != nil {
		writeErrorJSON(peer.conn, CodeProtocolError)
		return
	}

	// 验证配对码
	s.mu.Lock()
	session := s.sessions[pairFrame.SessionID]
	if session == nil {
		s.mu.Unlock()
		writeErrorJSON(peer.conn, CodeTargetUnavailable)
		return
	}

	if session.pairingSecret != pairFrame.PairingSecret {
		s.mu.Unlock()
		writeErrorJSON(peer.conn, CodePairingRejected)
		return
	}

	if time.Now().After(session.pairingExpiresAt) {
		s.mu.Unlock()
		writeErrorJSON(peer.conn, CodeTimeout)
		return
	}

	// 配对成功
	clientID := "mc_" + uuid.NewString()
	session.client = peer
	session.clientID = clientID
	s.mu.Unlock()

	// 发送配对成功响应
	pairedResp := ClientPairedFrame{
		Type:      TypeClientPaired,
		Version:   Version,
		SessionID: pairFrame.SessionID,
		ClientID:  clientID,
	}
	if err := peer.WriteJSON(pairedResp); err != nil {
		return
	}

	// 通知 agent 有 client 连接
	s.mu.Lock()
	if session.agent != nil {
		attachFrame := ClientAttachedFrame{
			Type:      TypeClientAttached,
			Version:   Version,
			SessionID: pairFrame.SessionID,
			ClientID:  clientID,
		}
		session.agent.WriteJSON(attachFrame)
	}
	s.mu.Unlock()

	// 开始转发消息
	s.forwardLoop(peer, pairFrame.SessionID, DirectionClientToAgent)
}

// forwardLoop 转发消息。
func (s *Server) forwardLoop(peer *peerConn, sessionID string, direction string) {
	for {
		_, raw, err := peer.conn.ReadMessage()
		if err != nil {
			return
		}

		// 解析帧
		var frame ControlFrame
		if err := json.Unmarshal(raw, &frame); err != nil {
			continue
		}

		// 处理 ping/pong
		if frame.Type == TypeRelayPong {
			continue
		}

		// 转发消息
		if frame.Type == TypeRelayForward {
			var env ForwardEnvelope
			if err := json.Unmarshal(raw, &env); err != nil {
				continue
			}

			// 查找目标连接
			s.mu.Lock()
			session := s.sessions[sessionID]
			if session == nil {
				s.mu.Unlock()
				continue
			}

			var target *peerConn
			if direction == DirectionAgentToClient {
				target = session.client
			} else {
				target = session.agent
			}
			s.mu.Unlock()

			if target != nil {
				target.WriteJSON(env)
			}
		}
	}
}

// markAgentDisconnected 标记 agent 断开连接。
func (s *Server) markAgentDisconnected(peer *peerConn) {
	for _, session := range s.sessions {
		if session.agent == peer {
			session.agent = nil
			session.agentDisconnectedAt = time.Now()
			// 通知 client agent 已断开
			if session.client != nil {
				errorFrame := NewErrorFrame(CodeAgentDisconnected)
				session.client.WriteJSON(errorFrame)
			}
			break
		}
	}
}

// markClientDisconnected 标记 client 断开连接。
func (s *Server) markClientDisconnected(peer *peerConn) {
	for _, session := range s.sessions {
		if session.client == peer {
			session.client = nil
			session.clientID = ""
			// 通知 agent client 已断开
			if session.agent != nil {
				errorFrame := NewErrorFrame(CodeTargetUnavailable)
				session.agent.WriteJSON(errorFrame)
			}
			break
		}
	}
}

// generateSecret 生成随机配对码。
func generateSecret() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// secretHash 计算配对码的哈希。
func secretHash(secret string) string {
	hash := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(hash[:])
}

// writeErrorJSON 写入错误帧。
func writeErrorJSON(conn *websocket.Conn, code string) {
	frame := NewErrorFrame(code)
	conn.WriteJSON(frame)
}

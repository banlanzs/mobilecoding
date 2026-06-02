package ws

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const readTimeout = 300 * time.Second
const writeTimeout = 10 * time.Second
const pingInterval = 15 * time.Second

type Conn struct {
	ws     *websocket.Conn
	send   chan Envelope
	closed chan struct{}
}

func NewConn(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(req *http.Request) bool {
			origin := req.Header.Get("Origin")
			if origin == "" {
				return true
			}
			host := req.Host
			return originMatchesHost(origin, host)
		},
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	conn := &Conn{
		ws:     c,
		send:   make(chan Envelope, 64),
		closed: make(chan struct{}),
	}
	c.SetPongHandler(func(string) error {
		c.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})
	go conn.writeLoop()
	go conn.pingLoop()
	return conn, nil
}

func (c *Conn) Send(env Envelope) bool {
	select {
	case c.send <- env:
		return true
	default:
		return false
	}
}

func (c *Conn) Read() (Envelope, bool) {
	c.ws.SetReadDeadline(time.Now().Add(readTimeout))
	var env Envelope
	if err := c.ws.ReadJSON(&env); err != nil {
		return Envelope{}, false
	}
	return env, true
}

func (c *Conn) Close() error {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
	return c.ws.Close()
}

func (c *Conn) writeLoop() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case env, ok := <-c.send:
			if !ok {
				return
			}
			c.ws.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.ws.WriteJSON(env); err != nil {
				return
			}
		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.closed:
			return
		}
	}
}

func (c *Conn) pingLoop() {}

func originMatchesHost(origin, host string) bool {
	if len(origin) < 8 {
		return false
	}
	switch {
	case len(origin) >= 7 && origin[:7] == "http://":
		origin = origin[7:]
	case len(origin) >= 8 && origin[:8] == "https://":
		origin = origin[8:]
	case len(origin) >= 6 && origin[:6] == "wss://":
		origin = origin[6:]
	case len(origin) >= 5 && origin[:5] == "ws://":
		origin = origin[5:]
	}
	for i := 0; i < len(origin); i++ {
		if origin[i] == '/' {
			origin = origin[:i]
			break
		}
	}
	return origin == host
}

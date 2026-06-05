package ws

import (
	"log"
	"sync"
	"sync/atomic"
)

type Hub struct {
	mu           sync.Mutex
	subscribers  map[chan Envelope]struct{}
	onConnect    func()  // 新连接回调（可选，用于 Local/Remote 切换通知）
	onDisconnect func()  // 连接断开回调（可选）
	dropCount    atomic.Int64 // 背压丢弃计数
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[chan Envelope]struct{})}
}

// SetCallbacks 设置连接/断开回调。
func (h *Hub) SetCallbacks(onConnect, onDisconnect func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onConnect = onConnect
	h.onDisconnect = onDisconnect
}

func (h *Hub) Subscribe() chan Envelope {
	ch := make(chan Envelope, 128)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	cb := h.onConnect
	h.mu.Unlock()
	if cb != nil {
		go cb()
	}
	return ch
}

func (h *Hub) Unsubscribe(ch chan Envelope) {
	h.mu.Lock()
	var disconnectCb func()
	if _, ok := h.subscribers[ch]; ok {
		delete(h.subscribers, ch)
		close(ch)
		if len(h.subscribers) == 0 && h.onDisconnect != nil {
			disconnectCb = h.onDisconnect
		}
	}
	h.mu.Unlock()
	if disconnectCb != nil {
		go disconnectCb()
	}
}

func (h *Hub) Broadcast(env Envelope) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subscribers {
		select {
		case ch <- env:
		default:
			drops := h.dropCount.Add(1)
			if drops%100 == 1 {
				log.Printf("[hub] backpressure: dropped %d messages total", drops)
			}
		}
	}
}

func (h *Hub) SubscriberCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subscribers)
}

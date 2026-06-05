package ws

import (
	"log"
	"sync"
	"sync/atomic"
)

const replayBufferSize = 200 // 最近 200 条事件用于 replay

type Hub struct {
	mu           sync.Mutex
	subscribers  map[chan Envelope]struct{}
	onConnect    func()  // 新连接回调（可选，用于 Local/Remote 切换通知）
	onDisconnect func()  // 连接断开回调（可选）
	dropCount    atomic.Int64 // 背压丢弃计数
	replayBuf    []Envelope   // replay 缓冲区（最近 N 条事件）
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

// Subscribe 订阅事件流。新连接会先收到 replay 缓冲区中的历史事件。
// bufferSize 为 0 时使用默认值 128。
func (h *Hub) Subscribe(bufferSize ...int) chan Envelope {
	size := 128
	if len(bufferSize) > 0 && bufferSize[0] > 0 {
		size = bufferSize[0]
	}
	ch := make(chan Envelope, size)
	h.mu.Lock()
	// Replay：将缓冲区中的历史事件发送给新订阅者
	for _, env := range h.replayBuf {
		select {
		case ch <- env:
		default:
			// 新订阅者缓冲区满，跳过旧事件
		}
	}
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
	// 维护 replay 缓冲区
	h.replayBuf = append(h.replayBuf, env)
	if len(h.replayBuf) > replayBufferSize {
		h.replayBuf = h.replayBuf[len(h.replayBuf)-replayBufferSize:]
	}
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

// ClearReplay 清空 replay 缓冲区（会话切换时调用）。
func (h *Hub) ClearReplay() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.replayBuf = nil
}

func (h *Hub) SubscriberCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subscribers)
}

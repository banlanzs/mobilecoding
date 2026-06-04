package ws

import "sync"

type Hub struct {
	mu          sync.Mutex
	subscribers map[chan Envelope]struct{}
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[chan Envelope]struct{})}
}

func (h *Hub) Subscribe() chan Envelope {
	ch := make(chan Envelope, 128)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(ch chan Envelope) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subscribers[ch]; ok {
		delete(h.subscribers, ch)
		close(ch)
	}
}

func (h *Hub) Broadcast(env Envelope) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subscribers {
		select {
		case ch <- env:
		default:
			// 背压：丢下一条
		}
	}
}

func (h *Hub) SubscriberCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subscribers)
}

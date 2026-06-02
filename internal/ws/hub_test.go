package ws

import (
	"sync"
	"testing"
	"time"
)

func TestHubBroadcast(t *testing.T) {
	h := NewHub()
	var wg sync.WaitGroup
	collected := make([]Envelope, 0)
	var mu sync.Mutex
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch := h.Subscribe()
			select {
			case env := <-ch:
				mu.Lock()
				collected = append(collected, env)
				mu.Unlock()
			case <-time.After(1 * time.Second):
			}
		}()
	}
	// 等所有订阅者注册
	time.Sleep(50 * time.Millisecond)
	h.Broadcast(Envelope{Type: "evt", SessionID: "sess_x"})
	wg.Wait()
	mu.Lock()
	defer mu.Unlock()
	if len(collected) != 3 {
		t.Errorf("collected = %d, want 3", len(collected))
	}
}

func TestHubUnsubscribeStopsDelivery(t *testing.T) {
	h := NewHub()
	ch := h.Subscribe()
	h.Unsubscribe(ch)
	h.Broadcast(Envelope{Type: "evt"})
	select {
	case _, ok := <-ch:
		if ok {
			t.Errorf("expected closed channel after unsubscribe")
		}
	case <-time.After(100 * time.Millisecond):
		// closed channel may already be drained; that's fine
	}
}

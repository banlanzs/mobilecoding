package projection

import (
	"strings"

	"github.com/google/uuid"

	"github.com/jaycrl/mytool/internal/engine"
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

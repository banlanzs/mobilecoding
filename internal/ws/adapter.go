package ws

import (
	"encoding/json"

	"github.com/banlanzs/mobilecoding/internal/projection"
)

// ProjectionToEnvelope 把 projection.Event 包装为 ws.Envelope（evt 类型）。
// 导出供 main.go 使用。
func ProjectionToEnvelope(p projection.Event) (Envelope, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{Type: "evt", SessionID: p.SessionID, Event: raw}, nil
}

// projectionToEnvelope 是内部别名，保持向后兼容。
func projectionToEnvelope(p projection.Event) (Envelope, error) {
	return ProjectionToEnvelope(p)
}

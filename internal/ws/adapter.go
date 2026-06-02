package ws

import (
	"encoding/json"

	"github.com/banlanzs/mobilecoding/internal/projection"
)

// projectionToEnvelope 把 projection.Event 包装为 ws.Envelope（evt 类型）。
func projectionToEnvelope(p projection.Event) (Envelope, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{Type: "evt", SessionID: p.SessionID, Event: raw}, nil
}

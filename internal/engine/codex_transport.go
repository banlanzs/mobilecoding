package engine

import (
	"encoding/json"
	"errors"
)

// ParseCodexJSONRPC 解析 codex app-server 的 JSON-RPC notification。
// 返回 Event{Kind: EventRaw, Data: <json bytes>}。
func ParseCodexJSONRPC(line []byte) (Event, error) {
	if len(line) == 0 {
		return Event{Kind: EventRaw, Data: nil}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(line, &m); err != nil {
		return Event{}, errors.New("invalid JSON: " + err.Error())
	}
	cp := make([]byte, len(line))
	copy(cp, line)
	return Event{Kind: EventRaw, Data: cp}, nil
}

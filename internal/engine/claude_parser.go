package engine

import (
	"encoding/json"
	"errors"
)

// ParseClaudeStreamJSON 解析 claude --output-format stream-json 的单行 JSON。
// 返回 Event{Kind: EventRaw, Data: <json bytes>}。
// MVP 2：不做深度解析，只透传 JSON；projection 层负责结构化。
func ParseClaudeStreamJSON(line []byte) (Event, error) {
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
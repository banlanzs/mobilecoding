package store

// SkillEntry 表示一个 skill。
type SkillEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ListSkills 列出工作区的 skills。
func ListSkills(workspace string) ([]SkillEntry, error) {
	// MVP 2：简单占位
	return nil, nil
}

package store

import (
	"os"
	"path/filepath"
	"strings"
)

// SkillEntry 表示一个 skill。
type SkillEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ListSkills 列出工作区的 skills。
func ListSkills(workspace string) ([]SkillEntry, error) {
	skillsDir := filepath.Join(workspace, ".claude", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SkillEntry{}, nil
		}
		return nil, err
	}
	var result []SkillEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		result = append(result, SkillEntry{
			Name: name,
			Path: filepath.Join(skillsDir, e.Name()),
		})
	}
	return result, nil
}

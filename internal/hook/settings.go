package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// SettingsInjector 负责在 Claude Code 的 settings.json 中注入/移除 HTTP hook。
// 设计目标：
//   - 幂等：多次调用结果相同
//   - 可逆：uninstall 时把原文件还原
//   - 安全：写入前自动备份到 settings.json.mobilecoding.bak
type SettingsInjector struct {
	mu           sync.Mutex
	settingsPath string
	backupPath   string
	marker       string // 在 hooks 列表中标记的 id
}

// NewSettingsInjector 构造注入器。path 通常是 ~/.claude/settings.json。
func NewSettingsInjector(path string) *SettingsInjector {
	return &SettingsInjector{
		settingsPath: path,
		backupPath:   path + ".mobilecoding.bak",
		marker:       "mobilecoding-hook",
	}
}

// DefaultSettingsPath 返回默认的 Claude settings.json 路径。
func DefaultSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home: %w", err)
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// ProjectSettingsPath 返回当前项目本地 Claude settings 路径。
func ProjectSettingsPath(cwd string) string {
	return filepath.Join(cwd, ".claude", "settings.local.json")
}

// HookConfig 描述要注入的 hook 配置。
type HookConfig struct {
	URL          string   // e.g. "http://127.0.0.1:8443/v1/hooks/permission-request"
	Token        string   // Bearer token, 通过 $MOBILECODING_TOKEN 注入
	Timeout      int      // seconds, 默认 300
	ExtraEnvVars []string // allowedEnvVars 中的额外环境变量
}

// Install 把 hook 配置写入 settings.json。
//   - 若文件不存在，创建一个只有 hooks 字段的最小 settings.json
//   - 若文件存在，解析后修改 hooks 字段
//   - 若 hook 已存在（marker 匹配），更新 URL/token 后写入
//   - 任何修改前都会先备份原文件到 .mobilecoding.bak
func (s *SettingsInjector) Install(cfg HookConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.settingsPath), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// 读现有文件
	original := map[string]any{}
	if data, err := os.ReadFile(s.settingsPath); err == nil {
		if err := json.Unmarshal(data, &original); err != nil {
			return fmt.Errorf("parse existing settings.json: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read settings.json: %w", err)
	}

	// 备份（仅备份一次：若 .bak 不存在）
	if _, err := os.Stat(s.backupPath); os.IsNotExist(err) {
		if data, err := json.MarshalIndent(original, "", "  "); err == nil {
			_ = os.WriteFile(s.backupPath, data, 0o644)
		}
	}

	// 修改 hooks.PermissionRequest
	updated := upsertHook(original, s.marker, cfg)

	// 写回
	out, err := json.MarshalIndent(updated, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(s.settingsPath, out, 0o644); err != nil {
		return fmt.Errorf("write settings.json: %w", err)
	}
	return nil
}

// RemoveInstalledHook 只移除 mobilecoding 标记的 hook，不还原备份。
func (s *SettingsInjector) RemoveInstalledHook() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	settings := map[string]any{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}
	removeHook(settings, s.marker)
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(s.settingsPath, out, 0o644)
}

// Uninstall 移除 hook 配置（保留文件中其他设置），并从备份还原。
//   - 如果存在备份文件，先尝试还原备份
//   - 如果没有备份，只移除我们的 marker 标识的 hook 条目
func (s *SettingsInjector) Uninstall() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 有备份则优先还原
	if _, err := os.Stat(s.backupPath); err == nil {
		data, err := os.ReadFile(s.backupPath)
		if err == nil {
			if err := os.WriteFile(s.settingsPath, data, 0o644); err != nil {
				return fmt.Errorf("restore from backup: %w", err)
			}
			_ = os.Remove(s.backupPath)
			return nil
		}
	}

	// 没有备份则只删除标记条目
	data, err := os.ReadFile(s.settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	settings := map[string]any{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}
	removeHook(settings, s.marker)
	out, _ := json.MarshalIndent(settings, "", "  ")
	out = append(out, '\n')
	return os.WriteFile(s.settingsPath, out, 0o644)
}

// IsInstalled 检查 hook 是否已注入（通过 marker 判断）。
func (s *SettingsInjector) IsInstalled() bool {
	data, err := os.ReadFile(s.settingsPath)
	if err != nil {
		return false
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return false
	}
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		return false
	}
	_, idx := findHookByMarker(hooks, s.marker)
	return idx >= 0
}

// upsertHook 在 settings 中找到 hooks.PermissionRequest 数组，
// 用 marker 标识 upsert 一条 mobilecoding 的 hook。
func upsertHook(settings map[string]any, marker string, cfg HookConfig) map[string]any {
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		settings["hooks"] = hooks
	}

	prList, _ := hooks["PermissionRequest"].([]any)
	if prList == nil {
		prList = []any{}
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 300
	}

	// 新的 hook entry
	newEntry := map[string]any{
		"matcher": "",
		"hooks": []any{
			map[string]any{
				"type":    "http",
				"url":     cfg.URL,
				"timeout": timeout,
				"headers": map[string]any{
					"Authorization": "Bearer " + cfg.Token,
				},
				"allowedEnvVars": append([]string{"MOBILECODING_TOKEN"}, cfg.ExtraEnvVars...),
				"_mobilecoding":  marker,
			},
		},
	}

	// 查找并替换
	replaced := false
	for i, item := range prList {
		m, _ := item.(map[string]any)
		if m == nil {
			continue
		}
		hs, _ := m["hooks"].([]any)
		for _, h := range hs {
			hm, _ := h.(map[string]any)
			if hm == nil {
				continue
			}
			if hm["_mobilecoding"] == marker {
				prList[i] = newEntry
				replaced = true
				break
			}
		}
		if replaced {
			break
		}
	}
	if !replaced {
		prList = append(prList, newEntry)
	}
	hooks["PermissionRequest"] = prList
	return settings
}

// removeHook 从 settings 中移除 marker 标识的 hook 条目。
func removeHook(settings map[string]any, marker string) {
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		return
	}
	for eventName, raw := range hooks {
		list, _ := raw.([]any)
		if list == nil {
			continue
		}
		filtered := make([]any, 0, len(list))
		for _, item := range list {
			m, _ := item.(map[string]any)
			if m == nil {
				filtered = append(filtered, item)
				continue
			}
			hs, _ := m["hooks"].([]any)
			hasMarker := false
			for _, h := range hs {
				hm, _ := h.(map[string]any)
				if hm != nil && hm["_mobilecoding"] == marker {
					hasMarker = true
					break
				}
			}
			if !hasMarker {
				filtered = append(filtered, item)
			}
		}
		hooks[eventName] = filtered
	}
}

// findHookByMarker 在 hooks map 中查找带 marker 的 entry。
// 返回 (eventName, index) 或 (nil, -1) 表示未找到。
func findHookByMarker(hooks map[string]any, marker string) (string, int) {
	for eventName, raw := range hooks {
		list, _ := raw.([]any)
		for i, item := range list {
			m, _ := item.(map[string]any)
			if m == nil {
				continue
			}
			hs, _ := m["hooks"].([]any)
			for _, h := range hs {
				hm, _ := h.(map[string]any)
				if hm != nil && hm["_mobilecoding"] == marker {
					return eventName, i
				}
			}
		}
	}
	return "", -1
}

// String 用于日志输出。
func (s *SettingsInjector) String() string {
	return fmt.Sprintf("SettingsInjector{path=%s, marker=%s}", s.settingsPath, s.marker)
}

var _ = strings.TrimSpace // 保留 strings import（备用）

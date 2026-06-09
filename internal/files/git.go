// Package files 提供文件系统和 git 操作。
package files

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GitFileStatus 表示单个文件的 git 状态。
type GitFileStatus struct {
	Path   string `json:"path"`
	Status string `json:"status"` // M/A/D/R/?? 等
	Staged bool   `json:"staged"`
}

// GitDiffSummary 表示 git diff 的摘要信息。
type GitDiffSummary struct {
	FilesChanged int    `json:"filesChanged"`
	Insertions   int    `json:"insertions"`
	Deletions    int    `json:"deletions"`
	Summary      string `json:"summary"`
}

// GetGitStatus 执行 git status --porcelain 并解析结果。
func GetGitStatus(cwd string) ([]GitFileStatus, error) {
	if cwd == "" {
		cwd = "."
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = cwd

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git status failed: %w (stderr: %s)", err, stderr.String())
	}

	var results []GitFileStatus
	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if len(line) < 4 {
			continue
		}

		// git status --porcelain 格式: XY PATH
		// X = staged, Y = unstaged
		// 例如: " M file.txt" (unstaged), "M  file.txt" (staged), "MM file.txt" (both)
		x := line[0]
		y := line[1]
		path := strings.TrimSpace(line[3:])

		// 确定状态
		var status string
		var staged bool

		if x != ' ' && x != '?' {
			// Staged change
			staged = true
			status = string(x)
		} else if y != ' ' && y != '?' {
			// Unstaged change
			staged = false
			status = string(y)
		} else if x == '?' && y == '?' {
			// Untracked
			staged = false
			status = "??"
		} else {
			// 未知状态，跳过
			continue
		}

		results = append(results, GitFileStatus{
			Path:   path,
			Status: status,
			Staged: staged,
		})
	}

	return results, nil
}

// GetGitDiff 获取指定文件的 diff 内容。
// filePath 为空时返回所有变更的 diff。
func GetGitDiff(cwd string, filePath string) (string, error) {
	if cwd == "" {
		cwd = "."
	}

	var args []string
	if filePath == "" {
		// 全部变更
		args = []string{"diff", "HEAD"}
	} else {
		// 特定文件
		args = []string{"diff", "HEAD", "--", filePath}
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = cwd

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// git diff 在没有变更时退出码为 0，有变更时也是 0
		// 只有真正出错时才非 0
		if stderr.Len() > 0 {
			return "", fmt.Errorf("git diff failed: %w (stderr: %s)", err, stderr.String())
		}
	}

	return stdout.String(), nil
}

// GetGitDiffSummary 获取 git diff 的统计摘要。
func GetGitDiffSummary(cwd string) (*GitDiffSummary, error) {
	if cwd == "" {
		cwd = "."
	}

	cmd := exec.Command("git", "diff", "HEAD", "--stat")
	cmd.Dir = cwd

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("git diff --stat failed: %w (stderr: %s)", err, stderr.String())
		}
	}

	output := stdout.String()
	if output == "" {
		return &GitDiffSummary{
			FilesChanged: 0,
			Insertions:   0,
			Deletions:    0,
			Summary:      "No changes",
		}, nil
	}

	// 解析最后一行的摘要
	// 格式: " 3 files changed, 45 insertions(+), 12 deletions(-)"
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return &GitDiffSummary{Summary: output}, nil
	}

	lastLine := lines[len(lines)-1]
	summary := &GitDiffSummary{Summary: lastLine}

	// 简单解析（可以改进为正则表达式）
	if strings.Contains(lastLine, "file") {
		parts := strings.Fields(lastLine)
		if len(parts) >= 2 {
			fmt.Sscanf(parts[0], "%d", &summary.FilesChanged)
		}
		for i, part := range parts {
			if strings.Contains(part, "insertion") && i > 0 {
				fmt.Sscanf(parts[i-1], "%d", &summary.Insertions)
			}
			if strings.Contains(part, "deletion") && i > 0 {
				fmt.Sscanf(parts[i-1], "%d", &summary.Deletions)
			}
		}
	}

	return summary, nil
}

//go:build !windows

// creack/pty only supports Unix-like platforms. On Windows this test file is
// excluded; production code paths are unchanged.

package engine

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// echoCommand 返回当前平台能用的"打印字符串并退出"命令与预期片段。
// Windows 下 Git Bash 自带 printf；其他 shell 使用 cmd /c echo。
func echoCommand(needle string) (cmd string, args []string, want string) {
	if runtime.GOOS == "windows" {
		// 优先用 Git Bash 自带 printf（若 PATH 中可解析）
		if _, err := exec.LookPath("printf"); err == nil {
			return "printf", []string{needle + "\n"}, needle
		}
		// 回退：cmd /c echo 自带 CRLF
		return "cmd", []string{"/c", "echo", needle}, needle
	}
	// Unix 类平台
	return "printf", []string{needle + "\n"}, needle
}

func TestPtyRunnerEcho(t *testing.T) {
	r := NewPtyRunner()
	needle := "hello-pty"
	cmd, args, want := echoCommand(needle)
	req := ExecRequest{
		Command: cmd,
		Args:    args,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.Start(ctx, req); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer r.Close()

	// 等到 stdout 出现目标字符串
	deadline := time.After(3 * time.Second)
	var got string
	for {
		select {
		case ev, ok := <-r.Events():
			if !ok {
				t.Fatalf("events channel closed before getting expected output")
			}
			if ev.Kind == EventRaw {
				got += string(ev.Data)
				if strings.Contains(got, want) {
					return // success
				}
			}
		case err := <-r.Errors():
			t.Fatalf("unexpected error: %v", err)
		case <-deadline:
			t.Fatalf("timeout waiting for output, got so far: %q", got)
		}
	}
}

func TestPtyRunnerStartRejectsEmptyCommand(t *testing.T) {
	r := NewPtyRunner()
	if err := r.Start(context.Background(), ExecRequest{Command: ""}); err == nil {
		t.Errorf("Start with empty command should fail")
	}
}

func TestPtyRunnerLifecycle(t *testing.T) {
	r := NewPtyRunner()
	req := ExecRequest{Command: "sleep", Args: []string{"0.3"}}
	if err := r.Start(context.Background(), req); err != nil {
		t.Fatalf("Start: %v", err)
	}
	// 给 50ms 启动时间
	time.Sleep(50 * time.Millisecond)
	select {
	case <-r.Done():
		t.Errorf("Done() should not be ready yet for a 0.3s sleep")
	default:
	}
	// 等到完成
	select {
	case <-r.Done():
	case <-time.After(2 * time.Second):
		t.Errorf("Done() should fire after sleep ends")
	}
	_ = r.Close()
}

func TestPtyRunnerDoneUnblocksOnConsumerLag(t *testing.T) {
	r := NewPtyRunner()
	req := ExecRequest{Command: "echo", Args: []string{"hi"}}
	if err := r.Start(context.Background(), req); err != nil {
		t.Fatalf("Start: %v", err)
	}
	// Intentionally do NOT drain Events() — simulate slow consumer.
	// Wait for the process to exit; Done() must still fire.
	select {
	case <-r.Done():
		// good
	case <-time.After(3 * time.Second):
		t.Errorf("Done() did not fire when consumer lagged; waitLoop likely blocked on full events channel")
	}
	_ = r.Close()
}

func TestPtyRunnerCloseIdempotent(t *testing.T) {
	r := NewPtyRunner()
	req := ExecRequest{Command: "echo", Args: []string{"hi"}}
	if err := r.Start(context.Background(), req); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

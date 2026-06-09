package store

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/banlanzs/mobilecoding/internal/projection"
)

func newTestStore(t *testing.T) *MessageStore {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSaveAndGetMessages(t *testing.T) {
	s := newTestStore(t)
	sid := "sess_test-1"

	ev1 := projection.Event{Type: projection.EventText, SessionID: sid, Time: time.Now(), Text: "hello", MessageID: "msg-1"}
	ev2 := projection.Event{Type: projection.EventText, SessionID: sid, Time: time.Now(), Text: "world", MessageID: "msg-2"}

	seq1, err := s.SaveMessage(sid, ev1)
	if err != nil {
		t.Fatal(err)
	}
	seq2, err := s.SaveMessage(sid, ev2)
	if err != nil {
		t.Fatal(err)
	}

	if seq1 != 1 {
		t.Errorf("seq1 = %d, want 1", seq1)
	}
	if seq2 != 2 {
		t.Errorf("seq2 = %d, want 2", seq2)
	}

	// GetMessagesAfter
	msgs, err := s.GetMessagesAfter(sid, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("len(msgs) = %d, want 2", len(msgs))
	}
	if msgs[0].Seq != 1 || msgs[1].Seq != 2 {
		t.Errorf("seq order wrong: %d, %d", msgs[0].Seq, msgs[1].Seq)
	}

	// GetMessagesAfter with afterSeq
	msgs, err = s.GetMessagesAfter(sid, 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	if msgs[0].Seq != 2 {
		t.Errorf("seq = %d, want 2", msgs[0].Seq)
	}
}

func TestGetMessagesBefore(t *testing.T) {
	s := newTestStore(t)
	sid := "sess_test-2"

	for i := 0; i < 5; i++ {
		ev := projection.Event{Type: projection.EventText, SessionID: sid, Time: time.Now(), MessageID: fmt.Sprintf("msg-%d", i)}
		if _, err := s.SaveMessage(sid, ev); err != nil {
			t.Fatal(err)
		}
	}

	// GetMessagesBefore
	msgs, err := s.GetMessagesBefore(sid, 4, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 3 {
		t.Fatalf("len(msgs) = %d, want 3", len(msgs))
	}
	// 应该是正序返回
	if msgs[0].Seq != 1 || msgs[2].Seq != 3 {
		t.Errorf("order wrong: %d, %d, %d", msgs[0].Seq, msgs[1].Seq, msgs[2].Seq)
	}
}

func TestGetLatestSeq(t *testing.T) {
	s := newTestStore(t)
	sid := "sess_test-3"

	seq, err := s.GetLatestSeq(sid)
	if err != nil {
		t.Fatal(err)
	}
	if seq != 0 {
		t.Errorf("seq = %d, want 0", seq)
	}

	ev := projection.Event{Type: projection.EventText, SessionID: sid, Time: time.Now()}
	if _, err := s.SaveMessage(sid, ev); err != nil {
		t.Fatal(err)
	}

	seq, err = s.GetLatestSeq(sid)
	if err != nil {
		t.Fatal(err)
	}
	if seq != 1 {
		t.Errorf("seq = %d, want 1", seq)
	}
}

func TestMultipleSessions(t *testing.T) {
	s := newTestStore(t)

	for _, sid := range []string{"sess_a", "sess_b"} {
		ev := projection.Event{Type: projection.EventText, SessionID: sid, Time: time.Now(), MessageID: sid + "-msg"}
		if _, err := s.SaveMessage(sid, ev); err != nil {
			t.Fatal(err)
		}
	}

	msgsA, _ := s.GetMessagesAfter("sess_a", 0, 100)
	msgsB, _ := s.GetMessagesAfter("sess_b", 0, 100)

	if len(msgsA) != 1 || len(msgsB) != 1 {
		t.Errorf("expected 1 message per session, got %d and %d", len(msgsA), len(msgsB))
	}
}

func TestCleanupOldSessions(t *testing.T) {
	s := newTestStore(t)

	// 插入一条消息
	ev := projection.Event{Type: projection.EventText, SessionID: "sess_old", Time: time.Now(), MessageID: "old-msg"}
	if _, err := s.SaveMessage("sess_old", ev); err != nil {
		t.Fatal(err)
	}

	// 清理 7 天前的消息（当前消息不到 1 天，不应被删除）
	deleted, err := s.CleanupOldSessions(7)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 0 {
		t.Errorf("deleted = %d, want 0", deleted)
	}

	// 清理 0 天前的消息（所有消息都会被删除）
	deleted, err = s.CleanupOldSessions(0)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}
}

func TestSeqMonotonicallyIncreasing(t *testing.T) {
	s := newTestStore(t)
	sid := "sess_mono"

	for i := 0; i < 100; i++ {
		ev := projection.Event{Type: projection.EventText, SessionID: sid, Time: time.Now(), MessageID: fmt.Sprintf("mono-%d", i)}
		seq, err := s.SaveMessage(sid, ev)
		if err != nil {
			t.Fatal(err)
		}
		if seq != int64(i+1) {
			t.Errorf("seq = %d, want %d", seq, i+1)
		}
	}
}

func TestOpenDBCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	s, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if _, err := os.Stat(dir); err != nil {
		t.Errorf("directory not created: %v", err)
	}
}

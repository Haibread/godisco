package health

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBeatThenCheck(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hb")
	now := time.Now()

	if err := Beat(path, now.Add(-10*time.Second), now); err != nil {
		t.Fatal(err)
	}
	if err := Check(path, now.Add(30*time.Second), MaxAge); err != nil {
		t.Errorf("fresh heartbeat should pass: %v", err)
	}
	if err := Check(path, now.Add(MaxAge+time.Second), MaxAge); err == nil {
		t.Error("heartbeat older than MaxAge should fail")
	}
}

func TestBeatSkipsStaleAck(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hb")
	now := time.Now()

	if err := Beat(path, now.Add(-staleAckAfter-time.Second), now); err != nil {
		t.Fatal(err)
	}
	if err := Check(path, now, MaxAge); err == nil {
		t.Error("a stale gateway ACK should not produce a heartbeat")
	}
}

func TestCheckMissingFile(t *testing.T) {
	if err := Check(filepath.Join(t.TempDir(), "absent"), time.Now(), MaxAge); err == nil {
		t.Error("missing heartbeat should fail")
	}
}

// Package health lets `godisco healthcheck`, run by the container runtime as a
// separate process, tell whether the running bot still hears from the gateway:
// the bot touches a file while heartbeats are acknowledged, the check reads its age.
package health

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// Discord asks for a heartbeat about every 41s; two missed ACKs mean the
	// gateway connection is gone even if discordgo has not noticed yet.
	staleAckAfter = 90 * time.Second
	beatEvery     = 15 * time.Second
	// MaxAge tolerates a few missed beats before the check fails.
	MaxAge = 60 * time.Second
)

// Path is where the running bot records its last healthy beat.
func Path() string {
	return filepath.Join(os.TempDir(), "godisco.heartbeat")
}

// Beat touches path when the last gateway ACK is recent enough, and leaves it
// to age otherwise.
func Beat(path string, lastAck, now time.Time) error {
	if now.Sub(lastAck) > staleAckAfter {
		return nil
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		return fmt.Errorf("write heartbeat: %w", err)
	}
	if err := os.Chtimes(path, now, now); err != nil {
		return fmt.Errorf("stamp heartbeat: %w", err)
	}
	return nil
}

// Check fails when path is missing or older than maxAge.
func Check(path string, now time.Time, maxAge time.Duration) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("no heartbeat: %w", err)
	}
	if age := now.Sub(info.ModTime()); age > maxAge {
		return fmt.Errorf("last heartbeat %s ago", age.Truncate(time.Second))
	}
	return nil
}

// Run beats until ctx is cancelled.
func Run(ctx context.Context, path string, lastAck func() time.Time, onErr func(error)) {
	ticker := time.NewTicker(beatEvery)
	defer ticker.Stop()
	for {
		if err := Beat(path, lastAck(), time.Now()); err != nil {
			onErr(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

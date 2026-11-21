package channels

import (
	"context"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// renameInterval is the period between full secondary-channel rename sweeps
// and the minimum time we let pass between renames of the same channel.
// Discord rate-limits channel renames to 2 per 10 minutes, so 5 minutes is
// the safe lower bound.
const renameInterval = 5 * time.Minute

var (
	lastRenameTimes = make(map[string]time.Time)
	lastRenameMu    sync.Mutex
)

// StartChannelLoops launches background loops that maintain managed channels.
// The returned WaitGroup is signalled when all loops have exited; cancelling
// ctx triggers shutdown.
func StartChannelLoops(ctx context.Context, s *discordgo.Session) *sync.WaitGroup {
	log.Info("Starting channel loops")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		secondaryChannelRenameLoop(ctx, s)
	}()
	return &wg
}

func secondaryChannelRenameLoop(ctx context.Context, s *discordgo.Session) {
	ticker := time.NewTicker(renameInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping secondary channel rename loop")
			return
		case <-ticker.C:
			log.Info("Checking for secondary channel rename")
			renameAllSecondaryChannels(s)
		}
	}
}

// renameSecondaryIfDue renames a single secondary channel if it hasn't been
// renamed in the last renameInterval. Both the periodic sweep and the
// presence-change handler funnel through here so they share one rate budget.
func renameSecondaryIfDue(s *discordgo.Session, channelID string) {
	lastRenameMu.Lock()
	last, hasLast := lastRenameTimes[channelID]
	if hasLast && time.Since(last) < renameInterval {
		lastRenameMu.Unlock()
		return
	}
	lastRenameTimes[channelID] = time.Now()
	lastRenameMu.Unlock()

	if err := renameOneSecondaryChannel(s, channelID); err != nil {
		log.Errorw("rename secondary channel", "channel_id", channelID, "error", err)
	}
}

// clearRenameThrottle drops the throttle entry for a deleted secondary
// channel so the map doesn't accumulate dead IDs.
func clearRenameThrottle(channelID string) {
	lastRenameMu.Lock()
	delete(lastRenameTimes, channelID)
	lastRenameMu.Unlock()
}

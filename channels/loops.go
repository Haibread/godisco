package channels

import (
	"context"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// renameInterval is the period between full secondary-channel rename sweeps.
// Discord rate-limits channel renames to 2 per 10 minutes, so 5 minutes is
// the safe lower bound.
const renameInterval = 5 * time.Minute

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

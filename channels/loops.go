package channels

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

func StartChannelLoops(s *discordgo.Session) {
	log.Info("Starting channel loops")
	secondaryChannelRename(s)
}

func secondaryChannelRename(s *discordgo.Session) {
	// Rename secondary channels every 5 minutes (Discord API limit)
	ticker := time.NewTicker(time.Second * 300)
	go func() {
		for range ticker.C {
			log.Info("Checking for secondary channel rename")
			renameAllSecondaryChannels(s)
		}
	}()
}

package channels

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

func VCUpdate(s *discordgo.Session, i *discordgo.VoiceStateUpdate) {
	if i.BeforeUpdate == nil {
		userJoined(s, i)

	} else if i.BeforeUpdate.ChannelID != "" && i.ChannelID != i.BeforeUpdate.ChannelID && i.ChannelID != "" {
		userMoved(s, i)

	} else if i.ChannelID == i.BeforeUpdate.ChannelID {
		return

	} else if i.ChannelID == "" {
		userMoved(s, i)
	}

}

// GuildCreate catches up on what happened while the bot was disconnected,
// since voice and channel events are not replayed on a new gateway session.
func GuildCreate(s *discordgo.Session, e *discordgo.GuildCreate) {
	// An outage GUILD_CREATE lists no channels: trusting it would forget every secondary.
	if e.Unavailable {
		log.Debugw("reconcile: guild unavailable, skipping", "guild_id", e.ID)
		return
	}

	before := time.Now()
	s.State.RLock()
	existing, occupied := liveChannels(e.Guild)
	s.State.RUnlock()

	reconcileSecondaries(e.ID, existing, occupied, before, func(channelID string) error {
		_, err := s.ChannelDelete(channelID)
		return err
	})
}

// ChannelDelete forgets a secondary channel removed from Discord, by the bot
// itself or by hand.
func ChannelDelete(s *discordgo.Session, e *discordgo.ChannelDelete) {
	if err := deleteSecondaryRecord(e.ID); err != nil {
		log.Errorw("channel delete: delete secondary record", "channel_id", e.ID, "error", err)
	}
}

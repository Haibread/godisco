package channels

import (
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

// PresenceUpdate triggers an immediate rename of the user's secondary
// channel when their game activity changes, so name updates feel live
// instead of waiting up to renameInterval for the next sweep. The actual
// rename is rate-limited per channel by renameSecondaryIfDue.
func PresenceUpdate(s *discordgo.Session, p *discordgo.PresenceUpdate) {
	if p == nil || p.User == nil || p.GuildID == "" {
		return
	}
	guild, err := s.State.Guild(p.GuildID)
	if err != nil {
		return
	}
	var channelID string
	for _, vs := range guild.VoiceStates {
		if vs.UserID == p.User.ID {
			channelID = vs.ChannelID
			break
		}
	}
	if channelID == "" {
		return
	}
	if !isChannelSecondary(s, channelID) {
		return
	}
	go renameSecondaryIfDue(s, channelID)
}

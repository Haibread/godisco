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

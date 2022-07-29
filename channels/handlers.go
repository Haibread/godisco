package channels

import (
	"github.com/bwmarrin/discordgo"
)

func VCUpdate(s *discordgo.Session, i *discordgo.VoiceStateUpdate) {
	if i.BeforeUpdate == nil {
		userJoined(s, i)

	} else if i.BeforeUpdate.ChannelID != "" && i.VoiceState.ChannelID != i.BeforeUpdate.ChannelID && i.VoiceState.ChannelID != "" {
		userMoved(s, i)

	} else if i.VoiceState.ChannelID == i.BeforeUpdate.ChannelID {
		return

	} else if i.VoiceState.ChannelID == "" {
		userMoved(s, i)
	}

}

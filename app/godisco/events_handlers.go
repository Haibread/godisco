package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func VCUpdate(s *discordgo.Session, i *discordgo.VoiceStateUpdate) {

	if i.BeforeUpdate == nil {
		fmt.Println("User Joined")
		userJoined(i)
	} else if i.BeforeUpdate.ChannelID != "" && i.VoiceState.ChannelID != i.BeforeUpdate.ChannelID && i.VoiceState.ChannelID != "" {
		fmt.Println("User moved")
	} else if i.VoiceState.ChannelID == i.BeforeUpdate.ChannelID {
		fmt.Println("User did something but did not move")
	} else if i.VoiceState.ChannelID == "" {
		fmt.Println("User left")
	}

}

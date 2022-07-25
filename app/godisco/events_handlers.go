package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func VCUpdate(s *discordgo.Session, i *discordgo.VoiceStateUpdate) {
	if i.BeforeUpdate == nil {
		User, err := s.User(i.UserID)
		if err != nil {
			log.Error(err)
		}
		fmt.Printf("User %v (%v) Joined channel %v", User.Username, i.UserID, i.ChannelID)
		userJoined(i)

	} else if i.BeforeUpdate.ChannelID != "" && i.VoiceState.ChannelID != i.BeforeUpdate.ChannelID && i.VoiceState.ChannelID != "" {
		userMoved(i)

	} else if i.VoiceState.ChannelID == i.BeforeUpdate.ChannelID {
		log.Debugf("User %v did something but nothing relevant happened", i.UserID)
		return

	} else if i.VoiceState.ChannelID == "" {
		userMoved(i)
	}

}

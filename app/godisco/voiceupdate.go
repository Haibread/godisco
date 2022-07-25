package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func userJoined(i *discordgo.VoiceStateUpdate) {

	if i.VoiceState.ChannelID == "941649245168091136" {
		fmt.Println("User joined the right channel")
		chanToCreate := &discordgo.GuildChannelCreateData{
			Name:     "Ow wow",
			Type:     discordgo.ChannelTypeGuildVoice,
			Bitrate:  96000,
			Position: 2,
			ParentID: "759133604554604574",
		}

		chanCreated, err := dg.GuildChannelCreateComplex("759083170619588669", *chanToCreate)
		if err != nil {
			log.Fatal(err)
		}

		//secondaryDB.AddChannel(chanCreated)
		fmt.Printf("Chan created : %+v", chanCreated)
	}
}

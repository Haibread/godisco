package main

import "github.com/bwmarrin/discordgo"

func ping(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseType(4),
		Data: &discordgo.InteractionResponseData{
			Content: "Pong",
		},
	})
}

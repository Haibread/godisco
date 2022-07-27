package commands

import "github.com/bwmarrin/discordgo"

func Ping(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseType(4),
		Data: &discordgo.InteractionResponseData{
			Content: "Pong",
		},
	})
}

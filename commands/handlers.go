package commands

import (
	"fmt"
	"time"

	"github.com/Haibread/godisco/channels"
	"github.com/bwmarrin/discordgo"
)

func Ping(s *discordgo.Session, i *discordgo.InteractionCreate) {
	messageTime, _ := discordgo.SnowflakeTimestamp(i.ID)
	delay := time.Since(messageTime)
	heartbeat := s.HeartbeatLatency()
	content := fmt.Sprintf("Pong! delay : %v, hearbeat : %vms", delay.Round(time.Millisecond), heartbeat)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

func Help(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Help isn't available yet, yeah I know, that sucks...",
		},
	})
}

func CreatePrimary(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userPerm, _ := s.State.UserChannelPermissions(i.User.ID, i.GuildID)
	if userPerm&discordgo.PermissionManageChannels != 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You don't have permission to do that.",
			},
		})
		return
	}

	// Get Variables
	content := ""
	options := i.ApplicationCommandData().Options

	defaultName, _ := options[0].Value.(string)
	template := options[1].Value.(string)

	// Check both var templating with fake data
	if err := channels.TestTemplate(s, defaultName); err != nil {
		content += "An error occured while testing the template"
	} else if err := channels.TestTemplate(s, template); err != nil {
		content += "An error occured while testing the template"
	} else {
		// Try to create channel
		if _, err := channels.CreatePrimaryChannel(s, i.GuildID, template, defaultName); err != nil {
			content += "An error occured while creating the channel"
		} else {
			content += fmt.Sprintf("Created primary with Default Name : '%s' and the template : '%s' \nYou can now change the name/settings/position... of the channel without any issue !", defaultName, template)
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

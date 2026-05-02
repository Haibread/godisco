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
	userPerm := i.Member.Permissions
	if userPerm&discordgo.PermissionManageChannels == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You don't have permission to do that.",
			},
		})
		return
	}

	options := i.ApplicationCommandData().Options
	defaultName, defaultNameOk := optionString(options, "default-name")
	template, templateOk := optionString(options, "template")

	content := ""
	switch {
	case !defaultNameOk || !templateOk:
		content = "Missing or invalid options."
	default:
		if err := channels.TestTemplate(s, defaultName); err != nil {
			content = "An error occured while testing the template"
		} else if err := channels.TestTemplate(s, template); err != nil {
			content = "An error occured while testing the template"
		} else if _, err := channels.CreatePrimaryChannel(s, i.GuildID, template, defaultName); err != nil {
			content = "An error occured while creating the channel"
		} else {
			content = fmt.Sprintf("Created primary with Default Name : '%s' and the template : '%s' \nYou can now change the name/settings/position... of the channel without any issue !", defaultName, template)
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

// optionString safely extracts a string option by name from a slash command's
// option list. Returns false if the option is missing or the wrong type.
func optionString(options []*discordgo.ApplicationCommandInteractionDataOption, name string) (string, bool) {
	for _, opt := range options {
		if opt.Name != name {
			continue
		}
		v, ok := opt.Value.(string)
		return v, ok
	}
	return "", false
}

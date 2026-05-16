package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

var (
	botCommands = []*discordgo.ApplicationCommand{
		{
			Type:        discordgo.ChatApplicationCommand,
			Name:        "ping",
			Description: "Show command and gateway latency",
		},
		{
			Type:        discordgo.ChatApplicationCommand,
			Name:        "help",
			Description: "Show commands and template-field reference",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "topic",
					Description: "Show help for a single topic",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "commands", Value: "commands"},
						{Name: "template", Value: "template"},
					},
				},
			},
		},
		{
			Type:        discordgo.ChatApplicationCommand,
			Name:        "create-primary",
			Description: "Create a new primary voice channel",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "default-name",
					Description: "Fallback name used when the template renders empty",
					Required:    true,
				},
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "template",
					Description:  "Go text/template string for the secondary channel name",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		{
			Type:        discordgo.ChatApplicationCommand,
			Name:        "list-primaries",
			Description: "List managed primary voice channels in this server",
		},
		{
			Type:        discordgo.ChatApplicationCommand,
			Name:        "delete-primary",
			Description: "Delete a managed primary voice channel",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionChannel,
					Name:         "channel",
					Description:  "The primary voice channel to delete",
					Required:     true,
					ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildVoice},
				},
			},
		},
	}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"ping":           Ping,
		"help":           Help,
		"create-primary": CreatePrimary,
		"list-primaries": ListPrimaries,
		"delete-primary": DeletePrimary,
	}
	autocompleteHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"create-primary": CreatePrimaryAutocomplete,
	}
)

// log is the package-level logger used by command handlers. It is set by
// RegisterCommands and must not be used before then.
var log *zap.SugaredLogger

func RegisterCommands(dg *discordgo.Session, l *zap.SugaredLogger) error {
	log = l
	if err := addCommands(dg, l); err != nil {
		return err
	}
	addHandlers(dg)
	return nil
}

func addCommands(dg *discordgo.Session, log *zap.SugaredLogger) error {
	log.Info("Adding commands")
	user, err := dg.User("@me")
	if err != nil {
		return fmt.Errorf("get bot user: %w", err)
	}
	if _, err := dg.ApplicationCommandBulkOverwrite(user.ID, "", botCommands); err != nil {
		return fmt.Errorf("bulk overwrite commands: %w", err)
	}
	return nil
}

func addHandlers(dg *discordgo.Session) {
	dg.AddHandler(
		func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			switch i.Type {
			case discordgo.InteractionApplicationCommand:
				if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
					h(s, i)
				}
			case discordgo.InteractionApplicationCommandAutocomplete:
				if h, ok := autocompleteHandlers[i.ApplicationCommandData().Name]; ok {
					h(s, i)
				}
			}
		})
}

func RemoveCommands(dg *discordgo.Session, log *zap.SugaredLogger) {
	applicationsCommandsAvailable, err := dg.ApplicationCommands(dg.State.User.ID, "")
	if err != nil {
		log.Errorf("Could not list application commands: %v", err)
		return
	}
	for _, v := range applicationsCommandsAvailable {
		if err = dg.ApplicationCommandDelete(dg.State.User.ID, "", v.ID); err != nil {
			log.Infof("Could not delete '%s' command: %v", v.Name, err)
			continue
		}
		log.Infof("Deleted command %s", v.Name)
	}
	log.Info("Deleted commands")
}

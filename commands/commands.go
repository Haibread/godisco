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
			Description: "Basic command",
		},
		{
			Type:        discordgo.ChatApplicationCommand,
			Name:        "help",
			Description: "Show commands help",
		},
		{
			Type:        discordgo.ChatApplicationCommand,
			Name:        "create-primary",
			Description: "Creates a new primary channel",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "default-name",
					Description: "The default name of a new secondary channel",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "template",
					Description: "The template of a new secondary channel",
					Required:    true,
				},
			},
		},
	}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"ping":           Ping,
		"help":           Help,
		"create-primary": CreatePrimary,
	}
)

func RegisterCommands(dg *discordgo.Session, log *zap.SugaredLogger) error {
	if err := addCommands(dg, log); err != nil {
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
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
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

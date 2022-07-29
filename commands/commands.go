package commands

import (
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
		"ping": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			Ping(s, i)
		},
		"help": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			Help(s, i)
		},
		"create-primary": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			CreatePrimary(s, i)
		},
	}
)

func RegisterCommands(dg *discordgo.Session, log *zap.SugaredLogger) {
	addCommands(dg, log)
	addHandlers(dg)
}

func addCommands(dg *discordgo.Session, log *zap.SugaredLogger) {
	log.Info("Adding commands")
	User, _ := dg.User("@me")
	UserID := User.ID
	_, err := dg.ApplicationCommandBulkOverwrite(UserID, "", botCommands)
	if err != nil {
		log.Panicf("Cannot create commands : %v", err)
	}
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
		log.Fatal(err)
	}
	for _, v := range applicationsCommandsAvailable {
		if err = dg.ApplicationCommandDelete(dg.State.User.ID, "", v.ID); err != nil {
			log.Infof("Could not delete '%s' command: %v", v.Name, err)
		}
		log.Infof("Deleted command %s", v.Name)
	}
	log.Info("Deleted commands")
}

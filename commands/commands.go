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
	}
	commandHandlerss = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"ping": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			Ping(s, i)
		},
	}
)

func RegisterCommands(dg *discordgo.Session, log *zap.SugaredLogger) {
	addCommands(dg, log)
	addHandlers(dg)
}

func addCommands(dg *discordgo.Session, log *zap.SugaredLogger) {
	log.Info("Adding commands")
	//_, err := dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", botCommands)
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
			if h, ok := commandHandlerss[i.ApplicationCommandData().Name]; ok {
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

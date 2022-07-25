package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Type:        discordgo.ChatApplicationCommand,
			Name:        "ping",
			Description: "Basic command",
		},
	}
	commandHandlerss = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"ping": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			ping(s, i)
		},
	}

	dg  *discordgo.Session
	log *zap.SugaredLogger
)

func main() {
	initLogger()
	initconfig()
	initDB()

	dg, err := discordgo.New("Bot " + viper.GetString("token"))
	if err != nil {
		log.Fatal("error creating discord session, ", err)
	}

	log.Info("Opening Websocket connection")
	err = dg.Open()
	if err != nil {
		log.Fatalf("Could not open Websocket connection %s", err)
	}

	dg.UpdateListeningStatus(viper.GetString("bot_status"))
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates | discordgo.IntentsGuildPresences

	log.Info("Adding handlers")
	dg.AddHandler(
		func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if h, ok := commandHandlerss[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		})
	dg.AddHandler(VCUpdate)

	//write new commands
	log.Info("Adding commands")
	_, err = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", commands)
	if err != nil {
		log.Panicf("Cannot create commands : %v", err)
	}

	applicationsCommandsAvailable, err := dg.ApplicationCommands(dg.State.User.ID, "")
	if err != nil {
		log.Fatal(err)
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Info("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Info("Starting to delete commands")
	// Commands delete
	for _, v := range applicationsCommandsAvailable {
		if err = dg.ApplicationCommandDelete(dg.State.User.ID, "", v.ID); err != nil {
			log.Infof("Could not delete '%s' command: %v", v.Name, err)
		}
		log.Infof("Deleted command %s", v.Name)
	}
	log.Info("Deleted commands")

	defer dg.Close()
}

func initLogger() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	log = logger.Sugar()
}

func initconfig() {
	viper.SetDefault("token", "")
	viper.SetDefault("bot_status", "Developped by Hybrid#0001")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
	})
	viper.WatchConfig()
}

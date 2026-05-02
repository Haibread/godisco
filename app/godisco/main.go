package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Haibread/godisco/channels"
	"github.com/Haibread/godisco/commands"
	"github.com/Haibread/godisco/database"
	"github.com/Haibread/godisco/logging"
	"github.com/bwmarrin/discordgo"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

func main() {
	logger, syncLogger := logging.InitLogger()
	defer syncLogger()

	channels.SetLogger(logger)
	database.SetLogger(logger)

	initconfig()
	database.InitDB()

	dg, err := discordgo.New("Bot " + viper.GetString("token"))
	if err != nil {
		logger.Fatal("error creating discord session, ", err)
	}

	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates | discordgo.IntentsGuildPresences

	logger.Info("Adding handlers")
	if err := commands.RegisterCommands(dg, logger); err != nil {
		logger.Fatalf("Failed to register commands: %v", err)
	}
	dg.AddHandler(channels.VCUpdate)

	logger.Info("Opening Websocket connection")
	err = dg.Open()
	if err != nil {
		logger.Fatalf("Could not open Websocket connection %s", err)
	}

	dg.UpdateListeningStatus(viper.GetString("bot_status"))

	channels.StartChannelLoops(dg)
	// Wait here until CTRL-C or other term signal is received.
	logger.Info("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	logger.Info("Shutting down")
	commands.RemoveCommands(dg, logger)
	dg.Close()
}

func initconfig() {
	viper.SetDefault("token", "")
	viper.SetDefault("bot_status", "")
	viper.SetConfigName("config")
	viper.AddConfigPath("config")
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

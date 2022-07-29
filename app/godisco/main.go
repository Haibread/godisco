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
	log := logging.InitLogger()
	initconfig()
	database.InitDB()

	dg, err := discordgo.New("Bot " + viper.GetString("token"))
	if err != nil {
		log.Fatal("error creating discord session, ", err)
	}

	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates | discordgo.IntentsGuildPresences

	log.Info("Adding handlers")
	commands.RegisterCommands(dg, log)
	dg.AddHandler(channels.VCUpdate)

	log.Info("Opening Websocket connection")
	err = dg.Open()
	if err != nil {
		log.Fatalf("Could not open Websocket connection %s", err)
	}

	dg.UpdateListeningStatus(viper.GetString("bot_status"))

	channels.StartChannelLoops(dg)
	// Wait here until CTRL-C or other term signal is received.
	log.Info("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	log.Info("Shutting down")
	// Commands delete
	commands.RemoveCommands(dg, log)
	dg.Close()
}

func initconfig() {
	viper.SetDefault("token", "")
	viper.SetDefault("bot_status", "Developped by Hybrid#0001")
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

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
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
	defer func() {
		// zap's Sync can return ENOTTY/EBADF on stderr in some environments;
		// nothing actionable beyond logging it.
		if err := syncLogger(); err != nil {
			logger.Debugw("logger sync", "error", err)
		}
	}()

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
	dg.AddHandler(channels.PresenceUpdate)

	logger.Info("Opening Websocket connection")
	err = dg.Open()
	if err != nil {
		logger.Fatalf("Could not open Websocket connection %s", err)
	}

	if status := viper.GetString("bot_status"); status != "" {
		if err := dg.UpdateStatusComplex(discordgo.UpdateStatusData{
			Activities: []*discordgo.Activity{{
				Name: status,
				Type: parseActivityType(viper.GetString("bot_activity_type")),
			}},
		}); err != nil {
			logger.Warnw("update bot status", "error", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	loopsDone := channels.StartChannelLoops(ctx, dg)

	logger.Info("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	logger.Info("Shutting down")
	cancel()
	loopsDone.Wait()
	commands.RemoveCommands(dg, logger)
	if err := dg.Close(); err != nil {
		logger.Warnw("close discord session", "error", err)
	}
}

func initconfig() {
	viper.SetDefault("token", "")
	viper.SetDefault("bot_status", "")
	viper.SetDefault("bot_activity_type", "Listening")
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

// parseActivityType maps the bot_activity_type config string to a discordgo
// ActivityType. Unknown values fall back to Listening to preserve the
// previous default.
func parseActivityType(s string) discordgo.ActivityType {
	switch strings.ToLower(s) {
	case "playing":
		return discordgo.ActivityTypeGame
	case "watching":
		return discordgo.ActivityTypeWatching
	case "competing":
		return discordgo.ActivityTypeCompeting
	case "streaming":
		return discordgo.ActivityTypeStreaming
	default:
		return discordgo.ActivityTypeListening
	}
}

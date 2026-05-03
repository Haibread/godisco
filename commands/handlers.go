package commands

import (
	"fmt"
	"time"

	"github.com/Haibread/godisco/channels"
	"github.com/bwmarrin/discordgo"
)

func Ping(s *discordgo.Session, i *discordgo.InteractionCreate) {
	messageTime, _ := discordgo.SnowflakeTimestamp(i.ID)
	delay := time.Since(messageTime).Round(time.Millisecond)
	heartbeat := s.HeartbeatLatency().Round(time.Millisecond)
	embed := &discordgo.MessageEmbed{
		Title: "Pong",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Command delay", Value: delay.String(), Inline: true},
			{Name: "Gateway heartbeat", Value: heartbeat.String(), Inline: true},
		},
	}
	respondEmbed(s, i, embed)
}

func Help(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "godisco",
		Description: "Manages dynamic voice channels. Join a primary channel to spawn your own secondary — it auto-deletes when empty.",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "Commands",
				Value: "`/ping` — command and gateway latency\n" +
					"`/help` — show this message\n" +
					"`/create-primary` — create a new primary voice channel (requires Manage Channels)",
			},
			{
				Name:  "Template fields",
				Value: channels.TemplateHelp,
			},
			{
				Name: "Examples",
				Value: "```\n" +
					"{{.Icao}} - {{.GameName}}              -> Alfa - Counter-Strike 2\n" +
					"#{{.Number}} {{.CreatorName}}'s room   -> #2 alice's room\n" +
					"```",
			},
		},
	}
	respondEmbed(s, i, embed)
}

func CreatePrimary(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Member.Permissions&discordgo.PermissionManageChannels == 0 {
		respond(s, i, "You don't have permission to do that.")
		return
	}

	options := i.ApplicationCommandData().Options
	defaultName, defaultNameOk := optionString(options, "default-name")
	template, templateOk := optionString(options, "template")
	if !defaultNameOk || !templateOk {
		respond(s, i, "Missing or invalid options.")
		return
	}

	if err := channels.TestTemplate(s, defaultName); err != nil {
		respond(s, i, fmt.Sprintf("Invalid `default-name` template: %v", err))
		return
	}
	if err := channels.TestTemplate(s, template); err != nil {
		respond(s, i, fmt.Sprintf("Invalid `template`: %v", err))
		return
	}
	if _, err := channels.CreatePrimaryChannel(s, i.GuildID, template, defaultName); err != nil {
		respond(s, i, fmt.Sprintf("Failed to create channel: %v", err))
		return
	}

	respondEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Primary channel created",
		Description: "You can rename, move, or change settings on the channel — godisco tracks it by ID.",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "default-name", Value: "`" + defaultName + "`", Inline: true},
			{Name: "template", Value: "`" + template + "`", Inline: true},
		},
	})
}

// respond sends an ephemeral text reply (visible only to the invoking user)
// and logs any error from the Discord API rather than letting it fall on the
// floor.
func respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	sendResponse(s, i, &discordgo.InteractionResponseData{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
}

// respondEmbed sends an ephemeral embed reply.
func respondEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	sendResponse(s, i, &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
}

func sendResponse(s *discordgo.Session, i *discordgo.InteractionCreate, data *discordgo.InteractionResponseData) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
	if err != nil && log != nil {
		log.Errorw("interaction respond",
			"interaction_id", i.ID,
			"command", i.ApplicationCommandData().Name,
			"error", err)
	}
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

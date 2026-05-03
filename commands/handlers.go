package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/Haibread/godisco/channels"
	"github.com/bwmarrin/discordgo"
)

// embedColor is the Discord blurple, applied to every godisco embed so
// replies are recognisable at a glance.
const embedColor = 0x5865F2

// templatePresets are offered as autocomplete suggestions on the
// /create-primary `template` option.
var templatePresets = []string{
	"{{.Icao}} - {{.GameName}}",
	"{{.Icao}} #{{.Number}}",
	"{{.CreatorName}}'s room",
	"#{{.Number}} {{.GameName}}",
	"#{{.Number}} {{.CreatorName}}",
	"{{.Icao}} - {{.CreatorName}}",
	"{{.GameName}} ({{.PartySize}})",
}

func Ping(s *discordgo.Session, i *discordgo.InteractionCreate) {
	messageTime, _ := discordgo.SnowflakeTimestamp(i.ID)
	delay := time.Since(messageTime).Round(time.Millisecond)
	heartbeat := s.HeartbeatLatency().Round(time.Millisecond)
	respondEmbed(s, i, &discordgo.MessageEmbed{
		Title: "Pong",
		Color: embedColor,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Command delay", Value: delay.String(), Inline: true},
			{Name: "Gateway heartbeat", Value: heartbeat.String(), Inline: true},
		},
	})
}

func Help(s *discordgo.Session, i *discordgo.InteractionCreate) {
	topic, _ := optionString(i.ApplicationCommandData().Options, "topic")
	respondEmbed(s, i, helpEmbed(topic))
}

func helpEmbed(topic string) *discordgo.MessageEmbed {
	commandsField := &discordgo.MessageEmbedField{
		Name: "Commands",
		Value: "`/ping` — command and gateway latency\n" +
			"`/help [topic]` — show this message or a single topic\n" +
			"`/create-primary` — create a new primary voice channel (Manage Channels)\n" +
			"`/list-primaries` — list managed primaries in this server\n" +
			"`/delete-primary` — delete a managed primary (Manage Channels)",
	}
	templateFields := &discordgo.MessageEmbedField{
		Name:  "Template fields",
		Value: channels.TemplateHelp,
	}
	examplesField := &discordgo.MessageEmbedField{
		Name: "Examples",
		Value: "```\n" +
			"{{.Icao}} - {{.GameName}}              -> Alfa - Counter-Strike 2\n" +
			"#{{.Number}} {{.CreatorName}}'s room   -> #2 alice's room\n" +
			"```",
	}

	switch topic {
	case "commands":
		return &discordgo.MessageEmbed{
			Title:  "godisco — commands",
			Color:  embedColor,
			Fields: []*discordgo.MessageEmbedField{commandsField},
		}
	case "template":
		return &discordgo.MessageEmbed{
			Title:  "godisco — channel-name templates",
			Color:  embedColor,
			Fields: []*discordgo.MessageEmbedField{templateFields, examplesField},
		}
	default:
		return &discordgo.MessageEmbed{
			Title:       "godisco",
			Color:       embedColor,
			Description: "Manages dynamic voice channels. Join a primary channel to spawn your own secondary — it auto-deletes when empty.",
			Fields:      []*discordgo.MessageEmbedField{commandsField, templateFields, examplesField},
		}
	}
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

	deferEphemeral(s, i)
	if _, err := channels.CreatePrimaryChannel(s, i.GuildID, template, defaultName); err != nil {
		editResponseContent(s, i, fmt.Sprintf("Failed to create channel: %v", err))
		return
	}

	editResponseEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Primary channel created",
		Color:       embedColor,
		Description: "You can rename, move, or change settings on the channel — godisco tracks it by ID.",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "default-name", Value: "`" + defaultName + "`", Inline: true},
			{Name: "template", Value: "`" + template + "`", Inline: true},
		},
	})
}

func ListPrimaries(s *discordgo.Session, i *discordgo.InteractionCreate) {
	primaries, err := channels.ListPrimarySummaries(i.GuildID)
	if err != nil {
		respond(s, i, fmt.Sprintf("Failed to list primaries: %v", err))
		return
	}
	if len(primaries) == 0 {
		respond(s, i, "No primary channels are registered in this server. Run `/create-primary` to add one.")
		return
	}

	fields := make([]*discordgo.MessageEmbedField, 0, len(primaries))
	for _, p := range primaries {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: fmt.Sprintf("<#%s>", p.ChannelID),
			Value: fmt.Sprintf("default-name: `%s`\ntemplate: `%s`\nactive secondaries: %d",
				p.DefaultName, p.Template, p.SecondaryCount),
		})
	}
	respondEmbed(s, i, &discordgo.MessageEmbed{
		Title:  fmt.Sprintf("Primary channels (%d)", len(primaries)),
		Color:  embedColor,
		Fields: fields,
	})
}

func DeletePrimary(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Member.Permissions&discordgo.PermissionManageChannels == 0 {
		respond(s, i, "You don't have permission to do that.")
		return
	}

	channelID, ok := optionString(i.ApplicationCommandData().Options, "channel")
	if !ok {
		respond(s, i, "Missing or invalid options.")
		return
	}

	deferEphemeral(s, i)
	if err := channels.DeletePrimaryChannel(s, i.GuildID, channelID); err != nil {
		editResponseContent(s, i, fmt.Sprintf("Failed to delete primary: %v", err))
		return
	}

	editResponseEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Primary channel deleted",
		Color:       embedColor,
		Description: fmt.Sprintf("Removed primary channel `%s` and its database record.", channelID),
	})
}

// CreatePrimaryAutocomplete suggests template presets as the user types
// the `template` option of /create-primary.
func CreatePrimaryAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var current, focusedName string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Focused {
			focusedName = opt.Name
			if v, ok := opt.Value.(string); ok {
				current = v
			}
			break
		}
	}
	if focusedName != "template" {
		sendAutocomplete(s, i, nil)
		return
	}

	needle := strings.ToLower(current)
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(templatePresets))
	for _, p := range templatePresets {
		if needle == "" || strings.Contains(strings.ToLower(p), needle) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  p,
				Value: p,
			})
			if len(choices) >= 25 {
				break
			}
		}
	}
	sendAutocomplete(s, i, choices)
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

// deferEphemeral acknowledges the interaction so the bot has up to 15
// minutes to finish slow work without Discord timing the user out at 3s.
func deferEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	})
	if err != nil && log != nil {
		log.Errorw("interaction defer",
			"interaction_id", i.ID,
			"command", i.ApplicationCommandData().Name,
			"error", err)
	}
}

func editResponseContent(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil && log != nil {
		log.Errorw("interaction edit", "interaction_id", i.ID, "error", err)
	}
}

func editResponseEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	embeds := []*discordgo.MessageEmbed{embed}
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Embeds: &embeds}); err != nil && log != nil {
		log.Errorw("interaction edit", "interaction_id", i.ID, "error", err)
	}
}

func sendAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate, choices []*discordgo.ApplicationCommandOptionChoice) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
	if err != nil && log != nil {
		log.Errorw("interaction autocomplete", "interaction_id", i.ID, "error", err)
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

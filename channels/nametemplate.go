package channels

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/Haibread/godisco/database"
	"github.com/Haibread/godisco/models"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ChanneltoRename struct {
	PrimaryChannel   *discordgo.Channel
	SecondaryChannel *discordgo.Channel
	Creator          string
	Rank             int
	Template         string
	templateVars     templateVars
	Logger           *zap.SugaredLogger
	Session          *discordgo.Session
}

type templateVars struct {
	Icao        string
	Number      string //rank (position from other secondaries channel)
	GameName    string //game activity name
	PartySize   string
	CreatorName string
}

var icao = [26]string{"Alfa", "Beta", "Charlie", "Delta", "Echo", "Foxtrot", "Golf", "Hotel", "India", "Juliett", "Kilo", "Lima", "Mike", "November", "Oscar", "Papa", "Quebec", "Romeo", "Sierra", "Tango", "Uniform", "Victor", "Whiskey", "X-ray", "Yankee", "Zulu"}

// TemplateHelp documents the fields available to channel-name templates. It
// is rendered inside a Discord embed by the /help command, so it uses a
// markdown code block for monospace alignment.
const TemplateHelp = "```\n" +
	"{{.Icao}}        NATO phonetic word for the channel rank\n" +
	"{{.Number}}      channel rank among siblings of the same primary\n" +
	"{{.GameName}}    most common active game in the channel\n" +
	"{{.PartySize}}   number of users currently in the channel\n" +
	"{{.CreatorName}} username of the channel creator\n" +
	"```"

func (c ChanneltoRename) getNamefromTemplate() (string, error) {
	vars := neededVariables(c.Template)
	for _, v := range vars {
		switch strings.ToLower(v) {
		case "icao":
			c.templateVars.Icao = getICAO(c.Rank)
		case "number":
			c.templateVars.Number = fmt.Sprintf("%d", c.Rank)
		case "gamename":
			game, err := c.resolveGameName()
			if err != nil {
				log.Errorw("resolve game name", "error", err)
				game = "Game Unknown"
			}
			c.templateVars.GameName = game
		case "partysize":
			c.templateVars.PartySize = c.resolvePartySize()
		case "creatorname":
			user, err := c.Session.User(c.Creator)
			if err != nil {
				log.Errorw("fetch creator user", "user_id", c.Creator, "error", err)
				continue
			}
			c.templateVars.CreatorName = user.Username
		}
	}

	templateName, err := template.New("channel_name").Parse(c.Template)
	if err != nil {
		return "", err
	}

	var tplOut bytes.Buffer
	if err := templateName.Execute(&tplOut, c.templateVars); err != nil {
		return "", err
	}
	return tplOut.String(), nil
}

// resolveGameName returns the game name to use for the current channel,
// falling back to the primary channel's default-name when no game can be
// detected. Returns "Game Unknown" only when neither is available.
func (c ChanneltoRename) resolveGameName() (string, error) {
	var game string

	switch {
	case c.PrimaryChannel != nil:
		user, err := c.Session.User(c.Creator)
		if err != nil {
			return "", fmt.Errorf("fetch creator: %w", err)
		}
		g, err := getMainActivityUser(c.Session, c.PrimaryChannel, user)
		if err != nil {
			return "", err
		}
		game = g
	case c.SecondaryChannel != nil:
		g, err := getMainActivity(c.Session, c.SecondaryChannel)
		if err != nil {
			return "", err
		}
		game = g
	default:
		return "Game Unknown", nil
	}

	if game != "" {
		return game, nil
	}

	parentID, err := c.parentChannelID()
	if err != nil {
		return "", err
	}
	if parentID == "" {
		return "", nil
	}

	fallback, err := getPrimaryChannelDefaultName(c.Session, parentID)
	if err != nil {
		return "", err
	}
	if fallback == "" {
		return "Game Unknown", nil
	}
	return fallback, nil
}

func (c ChanneltoRename) parentChannelID() (string, error) {
	if c.PrimaryChannel != nil {
		return c.PrimaryChannel.ID, nil
	}
	if c.SecondaryChannel == nil {
		return "", nil
	}
	var parent models.SecondaryChannel
	q := database.DB.Select("parent_channel_id").Where("channel_id = ?", c.SecondaryChannel.ID).First(&parent)
	if q.Error != nil {
		if errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("lookup parent of %s: %w", c.SecondaryChannel.ID, q.Error)
	}
	return parent.ParentChannelID, nil
}

// resolvePartySize counts users currently in the channel. For a brand-new
// secondary (PrimaryChannel set, SecondaryChannel nil) the size is 1: the
// user about to be moved into it.
func (c ChanneltoRename) resolvePartySize() string {
	if c.SecondaryChannel == nil {
		if c.PrimaryChannel != nil {
			return "1"
		}
		return "0"
	}
	users, err := getUsersInChannel(c.Session, c.SecondaryChannel)
	if err != nil {
		log.Errorw("count users in channel", "channel_id", c.SecondaryChannel.ID, "error", err)
		return "0"
	}
	return fmt.Sprintf("%d", len(users))
}

func neededVariables(template string) []string {
	toReturn := []string{}
	template_without_spaces := regexp.MustCompile(`\s+`).ReplaceAllString(template, "")
	r := regexp.MustCompile(`{{\.([^{}]*)}}`)
	matches := r.FindAllStringSubmatch(template_without_spaces, -1)
	for _, v := range matches {
		toReturn = append(toReturn, v[1])
	}
	return toReturn
}

// Test a template with fake data
func TestTemplate(s *discordgo.Session, tpl string) error {
	vars := &templateVars{
		Icao:        "Alfa",
		Number:      "1",
		GameName:    "Game Unknown",
		PartySize:   "14",
		CreatorName: "User",
	}

	templateName, err := template.New("test_template").Parse(tpl)
	if err != nil {
		return err
	}

	var tpl_out bytes.Buffer
	err = templateName.Execute(&tpl_out, vars)
	if err != nil {
		return err
	}

	return nil
}

func getICAO(position int) string {
	return icao[position]
}

func renameAllSecondaryChannels(s *discordgo.Session) {
	var channels []models.SecondaryChannel
	if err := database.DB.Find(&channels).Error; err != nil {
		log.Errorw("rename loop: list secondary channels", "error", err)
		return
	}
	for _, c := range channels {
		renameSecondaryIfDue(s, c.ChannelID)
	}
}

// renameOneSecondaryChannel performs the full rename pipeline for a single
// channel: look up its DB record, compute the templated name, and rename
// via the Discord API if it has changed.
func renameOneSecondaryChannel(s *discordgo.Session, channelID string) error {
	var record models.SecondaryChannel
	if err := database.DB.Where("channel_id = ?", channelID).First(&record).Error; err != nil {
		return fmt.Errorf("lookup secondary channel %s: %w", channelID, err)
	}

	parentChannel, err := s.State.Channel(record.ParentChannelID)
	if err != nil {
		return fmt.Errorf("fetch parent channel %s: %w", record.ParentChannelID, err)
	}
	secondaryChannel, err := s.State.Channel(channelID)
	if err != nil {
		return fmt.Errorf("fetch secondary channel: %w", err)
	}

	channelName, err := getChannelName(s, parentChannel, secondaryChannel, record.CreatorID)
	if err != nil {
		return fmt.Errorf("compute channel name: %w", err)
	}
	if channelName == secondaryChannel.Name {
		return nil
	}
	if _, err := s.ChannelEdit(channelID, &discordgo.ChannelEdit{Name: channelName}); err != nil {
		return fmt.Errorf("rename channel: %w", err)
	}
	return nil
}

func getUsersInChannel(s *discordgo.Session, channel *discordgo.Channel) ([]string, error) {
	//1. Get all users in channel
	guild, err := s.State.Guild(channel.GuildID)
	if err != nil {
		return []string{}, err
	}

	var users []string
	for _, c := range guild.VoiceStates {
		if c.ChannelID == channel.ID {
			users = append(users, c.UserID)
		}
	}

	//2. Return users
	return users, nil
}

func getUserPresence(s *discordgo.Session, GuildID string, UserID string) *discordgo.Presence {
	presence, err := s.State.Presence(GuildID, UserID)
	if err != nil {
		// Most often "presence not found" — only worth a debug line.
		log.Debugw("user presence lookup", "guild_id", GuildID, "user_id", UserID, "error", err)
		return nil
	}
	return presence
}

func getMainActivity(s *discordgo.Session, channel *discordgo.Channel) (string, error) {
	users, err := getUsersInChannel(s, channel)
	if err != nil {
		return "", err
	}

	var activity []string
	for _, user := range users {
		presence := getUserPresence(s, channel.GuildID, user)
		if presence == nil {
			continue
		}
		for _, p := range presence.Activities {
			if p.Type == discordgo.ActivityTypeGame || p.Type == discordgo.ActivityTypeCompeting {
				activity = append(activity, p.Name)
			}
		}
	}

	return mostCommon(activity), nil
}

func getMainActivityUser(s *discordgo.Session, channel *discordgo.Channel, User *discordgo.User) (string, error) {
	presence := getUserPresence(s, channel.GuildID, User.ID)
	if presence == nil {
		return "", nil
	}
	for _, p := range presence.Activities {
		if p.Type == discordgo.ActivityTypeGame || p.Type == discordgo.ActivityTypeCompeting {
			return p.Name, nil
		}
	}
	return "", nil
}

// mostCommon returns the most frequent string in items. Ties are broken
// by lexicographic order so the result is deterministic regardless of map
// iteration order.
func mostCommon(items []string) string {
	if len(items) == 0 {
		return ""
	}
	counts := make(map[string]int)
	for _, v := range items {
		counts[v]++
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] != counts[keys[j]] {
			return counts[keys[i]] > counts[keys[j]]
		}
		return keys[i] < keys[j]
	})
	return keys[0]
}

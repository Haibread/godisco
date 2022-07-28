package channels

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/Haibread/godisco/database"
	"github.com/Haibread/godisco/models"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
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

func RenameChannel(channel *discordgo.Channel, template string) {
	//TODO
	//1. Get channel name from template
	//2. If channel name is different from current name, rename channel
	//3. If channel name is the same, do nothing
}

func (c ChanneltoRename) getNamefromTemplate() (string, error) {
	vars := neededVariables(c.Template)
	for _, v := range vars {
		v = strings.ToLower(v)
		switch {
		case v == "icao":
			c.templateVars.Icao = getICAO(c.Rank)
		case v == "number":
			// We don't want 0 but we want 1
			c.templateVars.Number = fmt.Sprintf("%d", (c.Rank)+1)
		case v == "gamename":
			fmt.Println("Getting game name")
			// If primary channel
			if c.PrimaryChannel != nil {
				user, err := c.Session.User(c.Creator)
				if err != nil {
					log.Error(err)
					c.templateVars.GameName = "Game Unknown"
				}
				game, err := getMainActivityUser(c.Session, c.PrimaryChannel, user)
				if err != nil {
					log.Error(err)
					c.templateVars.GameName = "Game Unknown"
				}
				c.templateVars.GameName = game
			} else if c.SecondaryChannel != nil {
				game, err := getMainActivity(c.Session, c.SecondaryChannel)
				if err != nil {
					log.Error(err)
					c.templateVars.GameName = "Game Unknown"
				}
				c.templateVars.GameName = game
			} else {
				c.templateVars.GameName = "Game Unknown"
			}

		case v == "partysize":
			c.templateVars.PartySize = "N/A"
		case v == "creatorname":
			User, err := c.Session.User(c.Creator)
			if err != nil {
				log.Error(err)
			}
			c.templateVars.CreatorName = User.Username
		}
	}
	templateName, err := template.New("channel_name").Parse(c.Template)
	if err != nil {
		return "", err
	}

	var tpl_out bytes.Buffer
	err = templateName.Execute(&tpl_out, c.templateVars)
	if err != nil {
		return "", err
	}
	return tpl_out.String(), nil
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

func getICAO(position int) string {
	return icao[position]
}

func renameAllSecondaryChannels(s *discordgo.Session) {
	//1. Get all channels from db
	var channels []models.ManagedChannelCreated
	query := database.DB.Find(&channels)
	if query.Error != nil {
		log.Errorf("Failed to get all secondary channels %v", query.Error)
	}
	//2. For each channel, get name from template
	for _, c := range channels {
		fmt.Printf("secondary channel to maybe rename %+v\n", c)

		parentChannel, err := s.State.Channel(c.ParentChannelID)
		if err != nil {
			log.Error(err)
		}

		channelName, err := getChannelName(s, parentChannel, c.CreatorID)
		if err != nil {
			log.Error(err)
		}
		//3. If channel name is different from current name, rename channel
		currentChannel, err := s.State.Channel(c.ChannelID)
		currentChannelName := currentChannel.Name
		fmt.Printf("Wanted channel name: %s\n", channelName)
		fmt.Printf("Current channel name: %s\n", currentChannelName)
		if err != nil {
			log.Error(err)
		}
		if channelName != currentChannelName {
			s.ChannelEdit(c.ChannelID, channelName)
		}
		//4. If channel name is the same, do nothing
	}

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
		log.Error(err)
	}
	return presence
}

func getMainActivity(s *discordgo.Session, channel *discordgo.Channel) (string, error) {
	//1. Get all users in channel
	users, err := getUsersInChannel(s, channel)
	if err != nil {
		return "", err
	}

	//2. For each user, get presence
	var activity []string

	for _, user := range users {
		presence := getUserPresence(s, channel.GuildID, user)
		for _, p := range presence.Activities {
			if p.Type == discordgo.ActivityTypeGame || p.Type == discordgo.ActivityTypeCompeting {
				activity = append(activity, p.Name)
			}
		}
	}

	//3. Get most common activity
	duplicates := make(map[string]int)
	for _, v := range activity {
		// https://staticcheck.io/docs/checks#S1036
		duplicates[v] += 1
	}
	var mostCommon string
	var mostCommonCount int
	for k, v := range duplicates {
		if v > mostCommonCount {
			mostCommon = k
			mostCommonCount = v
		}
	}

	return mostCommon, nil
}

func getMainActivityUser(s *discordgo.Session, channel *discordgo.Channel, User *discordgo.User) (string, error) {
	presence := getUserPresence(s, channel.GuildID, User.ID)
	for _, p := range presence.Activities {
		if p.Type == discordgo.ActivityTypeGame || p.Type == discordgo.ActivityTypeCompeting {
			return p.Name, nil
		}
	}
	return "", nil
}

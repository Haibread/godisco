package channels

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type ChanneltoRename struct {
	ParentChannel *discordgo.Channel
	Creator       string
	Rank          int
	Template      string
	templateVars  templateVars
	Logger        *zap.SugaredLogger
	Session       *discordgo.Session
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
			c.templateVars.Number = fmt.Sprintf("%d", c.Rank)
		case v == "gamename":
			c.templateVars.GameName = "N/A"
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

func loopChannelsRename() {
	//TODO
	//1. Get all channels from db
	//2. For each channel, get name from template
	//3. If channel name is different from current name, rename channel
	//4. If channel name is the same, do nothing
}

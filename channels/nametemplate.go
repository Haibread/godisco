package channels

import (
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type Channel struct {
	Channel discordgo.Channel
	//ChannelID string
	//GuildID   string
	Template string
	Logger   *zap.SugaredLogger
	Session  *discordgo.Session
}

func (c Channel) GetChannelName() (channelName string, err error) {
	channel, err := c.Session.State.Channel(c.Channel.ID)
	channelName = channel.Name
	if err != nil {
		return "", err
	}
	return channelName, nil
}

func RenameChannel(channel *discordgo.Channel, template string) {
	//rename channel
}

func getNamefromTemplate(template string) string {
	//return channel name
	return ""
}

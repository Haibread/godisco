package channels

import (
	"errors"
	"fmt"

	"github.com/Haibread/godisco/database"
	"github.com/Haibread/godisco/logging"
	"github.com/Haibread/godisco/models"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var log *zap.SugaredLogger

func init() {
	log = logging.InitLogger()
}

func userJoined(s *discordgo.Session, i *discordgo.VoiceStateUpdate) {
	channel, err := s.State.Channel(i.ChannelID)
	if err != nil {
		log.Error(err)
		return
	}
	if isChannelManaged(s, i.VoiceState.ChannelID) {
		_, err := createChildChannelandMove(s, i, channel, i.VoiceState.UserID)
		//_, err := createChildChannel(channel)
		if err != nil {
			log.Error(err)
		}
	}

}

func createChildChannelandMove(s *discordgo.Session, i *discordgo.VoiceStateUpdate, parentChannel *discordgo.Channel, UserID string) (*discordgo.Channel, error) {

	channel, err := s.State.Channel(parentChannel.ID)
	if err != nil {
		return nil, err
	}

	createdChannel, err := createChildChannel(s, i, channel)
	if err != nil {
		return nil, err
	}

	err = s.GuildMemberMove(parentChannel.GuildID, UserID, &createdChannel.ID)
	if err != nil {
		return nil, err
	}

	return createdChannel, nil
}

func createChildChannel(s *discordgo.Session, i *discordgo.VoiceStateUpdate, parentChannel *discordgo.Channel) (*discordgo.Channel, error) {
	// Create data for new channel
	channelTemplateName, err := getManagedChannelTemplate(s, parentChannel.ID)
	if err != nil {
		channelTemplateName = "Managed Channel"
	}

	channelrank, err := getManagedChannelCreatedRank(s, parentChannel.ID)
	if err != nil {
		return nil, err
	}

	channel_tpl := &ChanneltoRename{
		ParentChannel: parentChannel,
		Creator:       i.UserID,
		Template:      channelTemplateName,
		Session:       s,
		Rank:          channelrank,
	}

	channelName, err := channel_tpl.getNamefromTemplate()
	if err != nil {
		return nil, err
	}

	channelToCreate := &discordgo.GuildChannelCreateData{
		Name:     channelName,
		Type:     discordgo.ChannelTypeGuildVoice,
		Bitrate:  parentChannel.Bitrate,
		Position: parentChannel.Position - 1,
		ParentID: parentChannel.ParentID,
	}

	// Create the new channel
	chanCreated, err := s.GuildChannelCreateComplex(parentChannel.GuildID, *channelToCreate)
	if err != nil {
		return nil, err
	}

	// Add channel in database.database.DB
	database.DB.Create(&models.ManagedChannelCreated{Name: chanCreated.Name, ChannelID: chanCreated.ID, GuildID: chanCreated.GuildID, ParentChannelID: parentChannel.ID})
	return chanCreated, nil
}

func userMoved(s *discordgo.Session, i *discordgo.VoiceStateUpdate) {
	// Check if the channel is managed
	if isChannelManaged(s, i.BeforeUpdate.ChannelID) {
		log.Debugf("Last channel %v was a primary channel, no action required", i.BeforeUpdate.ChannelID)
	} else if isChannelManaged(s, i.ChannelID) {
		log.Debugf("Current channel %v is a primary channel, we need to create a new channel", i.ChannelID)
		channel, err := s.Channel(i.ChannelID)
		if err != nil {
			log.Error(err)
			return
		}
		_, err = createChildChannelandMove(s, i, channel, i.UserID)
		if err != nil {
			log.Error(err)
		}
	}

	// Check if the channel is in managed channel created
	if isChannelManagedCreated(s, i.BeforeUpdate.ChannelID) {
		log.Debugf("Last channel %v was a secondary channel, checking if empty", i.BeforeUpdate.ChannelID)
		if isChannelEmpty(s, i.GuildID, i.BeforeUpdate.ChannelID) {
			log.Debugf("Secondary channel %v is empty on guild %v, deleting it", i.BeforeUpdate.ChannelID, i.GuildID)
			_, err := s.ChannelDelete(i.BeforeUpdate.ChannelID)
			if err != nil {
				log.Error(err)
			}
			log.Debugf("Removing secondary channel %v record from database.DB", i.BeforeUpdate.ChannelID)
			database.DB.Unscoped().Where("channel_id = ?", i.BeforeUpdate.ChannelID).Delete(&models.ManagedChannelCreated{})
		} else {
			log.Debugf("Secondary channel %v is not empty, no actions required", i.BeforeUpdate.ChannelID)
		}
	} else if isChannelManagedCreated(s, i.ChannelID) {
		log.Debugf("Current channel %v is a secondary channel, no actions required", i.ChannelID)
	}
}

func isChannelEmpty(s *discordgo.Session, GuildID string, ChannelID string) bool {
	count := 0
	guild, err := s.State.Guild(GuildID)
	if err != nil {
		log.Error(err)
	}

	for _, channel := range guild.VoiceStates {
		if channel.ChannelID == ChannelID {
			count += 1
		}
	}

	if count == 0 {
		return true
	} else {
		return false
	}
}

func isChannelManaged(s *discordgo.Session, ChannelID string) bool {
	var channel models.ManagedChannel

	managed_channel := database.DB.Select("channel_id").Where("channel_id = ?", ChannelID).First(&channel)

	if managed_channel.Error != nil {
		if errors.Is(managed_channel.Error, gorm.ErrRecordNotFound) {
			//log.Debugf("database.DB Record for Channel ID \"%v\" has not been found", ChannelID)
		} else {
			log.Error(managed_channel.Error)
		}
		return false
	}

	if channel.ChannelID != "" {
		return true
	}

	return false
}

func isChannelManagedCreated(s *discordgo.Session, ChannelID string) bool {
	var channel models.ManagedChannelCreated

	managed_channel := database.DB.Select("channel_id").Where("channel_id = ?", ChannelID).First(&channel)

	if managed_channel.Error != nil {
		if errors.Is(managed_channel.Error, gorm.ErrRecordNotFound) {
			//log.Debugf("database.DB Record for Channel ID \"%v\" has not been found", ChannelID)
		} else {
			log.Error(managed_channel.Error)
		}
		return false
	}

	if channel.ChannelID != "" {
		return true
	}

	return false
}

func getManagedChannelTemplate(s *discordgo.Session, ChannelID string) (string, error) {
	var channel models.ManagedChannel
	managed_channel := database.DB.Select("name_template").Where("channel_id = ?", ChannelID).First(&channel)

	if managed_channel.Error != nil {
		if errors.Is(managed_channel.Error, gorm.ErrRecordNotFound) {
			//log.Debugf("database.DB Record for Channel ID \"%v\" has not been found", ChannelID)
			return "", nil
		} else {
			return "", fmt.Errorf("error while getting channel template: %v", managed_channel.Error)
		}
	}

	if channel.NameTemplate != "" {
		return channel.NameTemplate, nil
	} else {
		return "", fmt.Errorf("no template found for channel %v", ChannelID)
	}
}

// Return the number of secondary channels already created
func getManagedChannelCreatedRank(s *discordgo.Session, ParentChannelID string) (int, error) {
	var count int64
	query := database.DB.Model(&models.ManagedChannelCreated{}).Where("parent_channel_id = ?", ParentChannelID).Count(&count)
	if query.Error != nil {
		if errors.Is(query.Error, gorm.ErrRecordNotFound) {
			return 0, nil
		} else {
			return 0, fmt.Errorf("error while getting secondary channels count : %v", query.Error)
		}
	}

	return int(count), nil
}

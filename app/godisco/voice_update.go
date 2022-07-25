package main

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

func userJoined(i *discordgo.VoiceStateUpdate) {
	channel, err := dg.Channel(i.ChannelID)
	if err != nil {
		log.Error(err)
		return
	}
	if isChannelManaged(i.ChannelID) {
		_, err = createChildChannelandMove(channel, i.UserID)
		if err != nil {
			log.Error(err)
		}
	}

}

func createChildChannelandMove(parentChannel *discordgo.Channel, User string) (*discordgo.Channel, error) {

	channel, err := dg.Channel(parentChannel.ID)
	if err != nil {
		return nil, err
	}

	createdChannel, err := createChildChannel(channel)
	if err != nil {
		return nil, err
	}

	err = dg.GuildMemberMove(parentChannel.GuildID, User, &createdChannel.ID)
	if err != nil {
		return nil, err
	}

	return createdChannel, nil
}

func createChildChannel(parentChannel *discordgo.Channel) (*discordgo.Channel, error) {
	// Create data for new channel
	channelTemplateName, err := getManagedChannelTemplate(parentChannel.ID)
	if err != nil {
		channelTemplateName = "Managed Channel"
	}

	channelName := fmt.Sprintf("%v-%v", channelTemplateName, rand.Intn(100))
	channelToCreate := &discordgo.GuildChannelCreateData{
		Name:     channelName,
		Type:     discordgo.ChannelTypeGuildVoice,
		Bitrate:  parentChannel.Bitrate,
		Position: parentChannel.Position - 1,
		ParentID: parentChannel.ParentID,
	}

	// Create the new channel
	chanCreated, err := dg.GuildChannelCreateComplex(parentChannel.GuildID, *channelToCreate)
	if err != nil {
		return nil, err
	}

	// Add channel in db
	db.Create(&ManagedChannelCreated{Name: chanCreated.Name, ChannelID: chanCreated.ID, GuildID: chanCreated.GuildID, ParentChannelID: parentChannel.ID})
	return chanCreated, nil
}

func userMoved(i *discordgo.VoiceStateUpdate) {
	// Check if the channel is managed
	if isChannelManaged(i.BeforeUpdate.ChannelID) {
		log.Debugf("Last channel %v is managed, no actions required", i.BeforeUpdate.ChannelID)
	} else if isChannelManaged(i.ChannelID) {
		log.Debugf("Current channel %v is managed, we need to create a new channel", i.ChannelID)
		channel, err := dg.Channel(i.ChannelID)
		if err != nil {
			log.Error(err)
			return
		}
		_, err = createChildChannelandMove(channel, i.UserID)
		if err != nil {
			log.Error(err)
		}
	}

	// Check if the channel is in managed channel created
	if isChannelManagedCreated(i.BeforeUpdate.ChannelID) {
		log.Debugf("Last channel %v is managed, no actions required", i.BeforeUpdate.ChannelID)
		if isChannelEmpty(i.GuildID, i.BeforeUpdate.ChannelID) {
			log.Debugf("Channel %v is empty on guild %v, deleting it", i.BeforeUpdate.ChannelID, i.GuildID)
			_, err := dg.ChannelDelete(i.BeforeUpdate.ChannelID)
			if err != nil {
				log.Error(err)
			}
			log.Debug("Removing channel record from db")
			db.Unscoped().Where("channel_id = ?", i.BeforeUpdate.ChannelID).Delete(&ManagedChannelCreated{})
		} else {
			log.Debugf("Channel %v is not empty, no actions required", i.BeforeUpdate.ChannelID)
		}
	} else if isChannelManagedCreated(i.ChannelID) {
		log.Debugf("Current channel %v is managed created, no actions required", i.ChannelID)
	}
}

func isChannelEmpty(GuildID string, ChannelID string) bool {
	count := 0
	guild, err := dg.State.Guild(GuildID)
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

func isChannelManaged(ChannelID string) bool {
	var channel ManagedChannel

	managed_channel := db.Select("channel_id").Where("channel_id = ?", ChannelID).First(&channel)

	if managed_channel.Error != nil {
		if errors.Is(managed_channel.Error, gorm.ErrRecordNotFound) {
			log.Debugf("DB Record for Channel ID \"%v\" has not been found", ChannelID)
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

func isChannelManagedCreated(ChannelID string) bool {
	var channel ManagedChannelCreated

	managed_channel := db.Select("channel_id").Where("channel_id = ?", ChannelID).First(&channel)

	if managed_channel.Error != nil {
		if errors.Is(managed_channel.Error, gorm.ErrRecordNotFound) {
			log.Debugf("DB Record for Channel ID \"%v\" has not been found", ChannelID)
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

func getManagedChannelTemplate(ChannelID string) (string, error) {
	var channel ManagedChannel
	managed_channel := db.Select("channel_id").Where("channel_id = ?", ChannelID).First(&channel)

	if managed_channel.Error != nil {
		if errors.Is(managed_channel.Error, gorm.ErrRecordNotFound) {
			log.Debugf("DB Record for Channel ID \"%v\" has not been found", ChannelID)
			return "", errors.New("channel not found in DB")
		} else {
			return "", fmt.Errorf("error while getting channel template: %v", managed_channel.Error)
		}
	}

	if channel.NameTemplate != "" {
		return channel.NameTemplate, nil
	} else {
		return "", errors.New("no template found")
	}
}

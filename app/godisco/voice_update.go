package main

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

func userJoined(i *discordgo.VoiceStateUpdate) {
	// Check if the channel is in db
	if !isChannelManaged(i.ChannelID) {
		return
	}

	// Get info from managedchannel
	channel, err := dg.Channel(i.ChannelID)
	if err != nil {
		log.Error(err)
		return
	}

	// Create data for new channel
	channelTemplateName, err := getManagedChannelTemplate(i.ChannelID)
	if err != nil {
		log.Error(err)
		channelTemplateName = "Managed Channel"
	}
	channelName := fmt.Sprintf("%v-%v", channelTemplateName, i.UserID)

	channelToCreate := &discordgo.GuildChannelCreateData{
		Name:     channelName,
		Type:     discordgo.ChannelTypeGuildVoice,
		Bitrate:  channel.Bitrate,
		Position: channel.Position,
		ParentID: channel.ParentID,
	}

	// Create the new channel
	chanCreated, err := dg.GuildChannelCreateComplex(i.GuildID, *channelToCreate)
	if err != nil {
		log.Error(err)
	}

	// Add channel in db
	db.Create(&ManagedChannelCreated{Name: chanCreated.Name, ChannelID: chanCreated.ID, GuildID: chanCreated.GuildID})

	// Move user to new channel
	err = dg.GuildMemberMove(i.GuildID, i.UserID, &chanCreated.ID)
	if err != nil {
		log.Error(err)
	}
}

func userMoved(i *discordgo.VoiceStateUpdate) {
	// Check if the channel is managed
	if isChannelManaged(i.BeforeUpdate.ChannelID) {
		fmt.Println("Last channel is managed, no actions required")
	} else if isChannelManaged(i.ChannelID) {
		fmt.Println("Current channel is managed, we need to create a new channel")
		userJoined(i)
	}

	// Check if the channel is in managed channel created
	if isChannelManagedCreated(i.BeforeUpdate.ChannelID) {
		fmt.Println("Last channel is managed created, checking if empty")
		if isChannelEmpty(i.GuildID, i.BeforeUpdate.ChannelID) {
			fmt.Println("Channel is empty, deleting it")
			_, err := dg.ChannelDelete(i.BeforeUpdate.ChannelID)
			if err != nil {
				log.Error(err)
			}
			log.Debug("Removing channel record from db")
			db.Unscoped().Where("channel_id = ?", i.BeforeUpdate.ChannelID).Delete(&ManagedChannelCreated{})
		}
	} else if isChannelManagedCreated(i.ChannelID) {
		fmt.Println("Current channel is managed created, no actions required")
	}
}

func isChannelEmpty(GuildID string, ChannelID string) bool {
	count := 0
	// Check if the channel is empty
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
			log.Error(managed_channel.Error)
			return "", fmt.Errorf("error while getting channel template: %v", managed_channel.Error)
		}
	}

	if channel.NameTemplate != "" {
		return channel.NameTemplate, nil
	} else {
		return "", errors.New("no template found")
	}
}

package main

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

func userJoined(i *discordgo.VoiceStateUpdate) {
	// Check if the channel is in db
	managed_channel := db.Select("channel_id").Where("channel_id = ?", i.ChannelID).First(&ManagedChannel{})
	if managed_channel.Error != nil {
		if errors.Is(managed_channel.Error, gorm.ErrRecordNotFound) {
			fmt.Printf("Channel %v is not managed", i.ChannelID)
		} else {
			log.Error(managed_channel.Error)
		}
		return
	}

	// Get info from managedchannel
	channel, err := dg.Channel(i.ChannelID)
	if err != nil {
		log.Error(err)
		return
	}

	// Create data for new channel
	channelToCreate := &discordgo.GuildChannelCreateData{
		Name:     "Managedbygodisco",
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
	var mc_result ManagedChannel
	managed_channel := db.Select("channel_id").Where("channel_id = ?", i.BeforeUpdate.ChannelID).First(&mc_result)
	if managed_channel.Error != nil {
		if errors.Is(managed_channel.Error, gorm.ErrRecordNotFound) {
			fmt.Printf("Channel %v is not managed\n", i.BeforeUpdate.ChannelID)
		} else {
			log.Error(managed_channel.Error)
			return
		}
	}

	if mc_result.ChannelID != "" {
		fmt.Println("Detect managed channel")
		// TEMP
		userJoined(i)
		return
	}

	fmt.Println("Reached beacon")
	// Check if the channel is in managed channel created
	var mch_result ManagedChannelCreated
	managed_channel_created := db.Select("channel_id").Where("channel_id = ?", i.BeforeUpdate.ChannelID).First(&mch_result)
	if managed_channel_created.Error != nil {
		if errors.Is(managed_channel_created.Error, gorm.ErrRecordNotFound) {
			fmt.Printf("Channel %v is not managed created\n", i.BeforeUpdate.ChannelID)
		} else {
			log.Error(managed_channel_created.Error)
		}
		return
	}

	if mch_result.ChannelID != "" {
		fmt.Println("Detect managed channel created")
	}

	fmt.Println("Reached second beacon")

	// Check if the channel is empty
	guild, err := dg.State.Guild(i.GuildID)
	if err != nil {
		log.Error(err)
	}

	count := 0
	for _, channel := range guild.VoiceStates {
		if channel.ChannelID == i.BeforeUpdate.ChannelID {
			count += 1
		}
	}

	fmt.Println("Reached third beacon")

	if count == 0 {
		// Delete channel
		_, err := dg.ChannelDelete(i.BeforeUpdate.ChannelID)
		if err != nil {
			log.Error(err)
		}
		// Delete channel from db
		db.Unscoped().Where("channel_id = ?", i.BeforeUpdate.ChannelID).Delete(&ManagedChannelCreated{})
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
	return false
}

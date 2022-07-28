package models

import "gorm.io/gorm"

type ManagedChannel struct {
	gorm.Model
	NameTemplate string `json:"name_template"`
	NameDefault  string `json:"name_default"`
	ChannelID    string `json:"channel_id"`
	GuildID      string `json:"guild_id"`
}

type ManagedChannelCreated struct {
	gorm.Model
	Name            string `json:"name"`
	ChannelID       string `json:"channel_id"`
	GuildID         string `json:"guild_id"`
	ParentChannelID string `json:"parent_channel_id"`
}

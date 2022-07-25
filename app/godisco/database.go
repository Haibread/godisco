package main

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ManagedChannel struct {
	gorm.Model
	NameTemplate string `json:"name_template"`
	ChannelID    string `json:"channel_id"`
	GuildID      string `json:"guild_id"`
}

type ManagedChannelCreated struct {
	gorm.Model
	Name      string `json:"name"`
	ChannelID string `json:"channel_id"`
	GuildID   string `json:"guild_id"`
}

var db *gorm.DB

func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal("Failed to connect to the database")
	}

	db.AutoMigrate(&ManagedChannel{})
	db.AutoMigrate(&ManagedChannelCreated{})
	db.FirstOrCreate(&ManagedChannel{ChannelID: "941649245168091136", GuildID: "759083170619588669"})
}

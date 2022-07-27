package database

import (
	"log"

	"github.com/Haibread/godisco/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() {
	var err error
	DB, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal("Failed to connect to the database")
	}

	DB.AutoMigrate(&models.ManagedChannel{})
	DB.AutoMigrate(&models.ManagedChannelCreated{})
	DB.FirstOrCreate(&models.ManagedChannel{ChannelID: "941649245168091136", GuildID: "759083170619588669"})
}

func GetDB() *gorm.DB {
	return DB
}

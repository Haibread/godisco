package database

import (
	"io/ioutil"

	"github.com/Haibread/godisco/logging"
	"github.com/Haibread/godisco/models"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB
var log *zap.SugaredLogger

func init() {
	log = logging.InitLogger()
}

func InitDB() {
	var err error
	createDBifNotExists()
	DB, err = gorm.Open(sqlite.Open("./config/channels.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		//Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to the database, err %v", err)
	}

	DB.AutoMigrate(&models.PrimaryChannel{})
	DB.AutoMigrate(&models.SecondaryChannel{})

	//Test data
	log.Info("Creating db entries")
	//var nameTemplate string = "{{.Icao}} {{.GameName}}"
	var nameTemplate string = "{{.GameName}}"
	DB.FirstOrCreate(&models.PrimaryChannel{ChannelID: "941649245168091136", GuildID: "759083170619588669", NameTemplate: nameTemplate, NameDefault: "Général"})
}

func GetDB() *gorm.DB {
	return DB
}

func createDBifNotExists() {
	if _, err := ioutil.ReadFile("./config/channels.db"); err != nil {
		log.Info("Creating db")
	}
}

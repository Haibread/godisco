package database

import (
	"errors"
	"io/fs"
	"os"

	"github.com/Haibread/godisco/models"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB
var log *zap.SugaredLogger

// SetLogger injects the logger used by this package. Must be called before InitDB.
func SetLogger(l *zap.SugaredLogger) {
	log = l
}

func InitDB() {
	var err error
	createDBifNotExists()
	DB, err = gorm.Open(sqlite.Open("./config/channels.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to connect to the database, err %v", err)
	}

	if err := DB.AutoMigrate(&models.PrimaryChannel{}); err != nil {
		log.Fatalf("Failed to migrate PrimaryChannel: %v", err)
	}
	if err := DB.AutoMigrate(&models.SecondaryChannel{}); err != nil {
		log.Fatalf("Failed to migrate SecondaryChannel: %v", err)
	}
}

func GetDB() *gorm.DB {
	return DB
}

func createDBifNotExists() {
	if _, err := os.Stat("./config/channels.db"); errors.Is(err, fs.ErrNotExist) {
		log.Info("Creating db")
	}
}

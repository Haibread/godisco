package database

import (
	"errors"
	"io/fs"
	"os"

	"github.com/Haibread/godisco/models"
	"github.com/ncruces/go-sqlite3/gormlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB
var log *zap.SugaredLogger

// SetLogger injects the logger used by this package. Must be called before InitDB.
func SetLogger(l *zap.SugaredLogger) {
	log = l
}

// InitDB opens the SQLite database at path, creating it if needed, and
// migrates it.
func InitDB(path string) {
	var err error
	createDBifNotExists(path)
	DB, err = gorm.Open(gormlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalw("Failed to connect to the database", "path", path, "error", err)
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

func createDBifNotExists(path string) {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		log.Infow("Creating db", "path", path)
	}
}

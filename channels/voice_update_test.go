package channels

import (
	"testing"

	"github.com/Haibread/godisco/database"
	"github.com/Haibread/godisco/models"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// setupTestDB wires database.DB to a fresh in-memory SQLite database with
// the project's models migrated. It restores the previous DB on cleanup.
func setupTestDB(t *testing.T) {
	t.Helper()

	log = zap.NewNop().Sugar()

	prev := database.DB
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.PrimaryChannel{}, &models.SecondaryChannel{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database.DB = db
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
		database.DB = prev
	})
}

func TestGetSecondaryChannelRank_NoSiblings(t *testing.T) {
	setupTestDB(t)

	// No existing secondary channels: a brand new channel should be rank 1.
	rank, err := getSecondaryChannelRank(nil, "parent-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rank != 1 {
		t.Errorf("rank = %d, want 1", rank)
	}
}

func TestGetSecondaryChannelRank_OrdersByChannelID(t *testing.T) {
	setupTestDB(t)

	// Insert siblings under the same parent. Channel IDs sort numerically,
	// so "100" < "200" < "300".
	siblings := []string{"300", "100", "200"}
	for _, id := range siblings {
		if err := database.DB.Create(&models.SecondaryChannel{
			ChannelID:       id,
			ParentChannelID: "parent-1",
		}).Error; err != nil {
			t.Fatalf("seed channel %s: %v", id, err)
		}
	}

	tests := map[string]int{
		"100": 1,
		"200": 2,
		"300": 3,
	}
	for id, want := range tests {
		got, err := getSecondaryChannelRank(nil, "parent-1", id)
		if err != nil {
			t.Fatalf("rank for %s: %v", id, err)
		}
		if got != want {
			t.Errorf("rank for channel %s = %d, want %d", id, got, want)
		}
	}
}

func TestGetSecondaryChannelRank_NewChannelGetsNextRank(t *testing.T) {
	setupTestDB(t)

	for _, id := range []string{"100", "200"} {
		if err := database.DB.Create(&models.SecondaryChannel{
			ChannelID:       id,
			ParentChannelID: "parent-1",
		}).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	// Empty ChannelID means "we are about to create a new one"; it should
	// be ranked after all existing siblings.
	rank, err := getSecondaryChannelRank(nil, "parent-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rank != 3 {
		t.Errorf("rank = %d, want 3", rank)
	}
}

func TestGetSecondaryChannelRank_IgnoresOtherParents(t *testing.T) {
	setupTestDB(t)

	if err := database.DB.Create(&models.SecondaryChannel{
		ChannelID:       "999",
		ParentChannelID: "other-parent",
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	rank, err := getSecondaryChannelRank(nil, "parent-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rank != 1 {
		t.Errorf("rank = %d, want 1 (siblings under other-parent should be ignored)", rank)
	}
}

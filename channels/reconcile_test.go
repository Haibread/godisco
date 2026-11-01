package channels

import (
	"errors"
	"testing"
	"time"

	"github.com/Haibread/godisco/database"
	"github.com/Haibread/godisco/models"
	"github.com/bwmarrin/discordgo"
)

const (
	testGuild  = "guild-1"
	testParent = "parent-1"
)

func seedSecondary(t *testing.T, channelID string) {
	t.Helper()
	if err := database.DB.Create(&models.SecondaryChannel{
		ChannelID:       channelID,
		GuildID:         testGuild,
		ParentChannelID: testParent,
	}).Error; err != nil {
		t.Fatalf("seed channel %s: %v", channelID, err)
	}
}

func hasSecondary(t *testing.T, channelID string) bool {
	t.Helper()
	var count int64
	if err := database.DB.Model(&models.SecondaryChannel{}).Where("channel_id = ?", channelID).Count(&count).Error; err != nil {
		t.Fatalf("count channel %s: %v", channelID, err)
	}
	return count > 0
}

// inFuture marks every seeded record as older than the reconciliation.
func inFuture() time.Time {
	return time.Now().Add(time.Minute)
}

func failOnDelete(t *testing.T) func(string) error {
	t.Helper()
	return func(channelID string) error {
		t.Errorf("unexpected delete of channel %s", channelID)
		return nil
	}
}

func TestReconcile_MissingChannelDropsRecordOnly(t *testing.T) {
	setupTestDB(t)
	seedSecondary(t, "100")

	reconcileSecondaries(testGuild, map[string]bool{}, map[string]bool{}, inFuture(), failOnDelete(t))

	if hasSecondary(t, "100") {
		t.Error("record of a channel missing from Discord should be dropped")
	}
}

func TestReconcile_GhostNoLongerShiftsRank(t *testing.T) {
	setupTestDB(t)
	seedSecondary(t, "100")

	reconcileSecondaries(testGuild, map[string]bool{}, map[string]bool{}, inFuture(), failOnDelete(t))

	rank, err := getSecondaryChannelRank(nil, testParent, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rank != 1 {
		t.Errorf("rank = %d, want 1", rank)
	}
}

func TestReconcile_EmptyChannelIsDeleted(t *testing.T) {
	setupTestDB(t)
	seedSecondary(t, "100")

	var deleted []string
	reconcileSecondaries(testGuild, map[string]bool{"100": true}, map[string]bool{}, inFuture(), func(channelID string) error {
		deleted = append(deleted, channelID)
		return nil
	})

	if len(deleted) != 1 || deleted[0] != "100" {
		t.Errorf("deleted = %v, want [100]", deleted)
	}
	if hasSecondary(t, "100") {
		t.Error("record of a deleted channel should be dropped")
	}
}

func TestReconcile_OccupiedChannelIsKept(t *testing.T) {
	setupTestDB(t)
	seedSecondary(t, "100")

	reconcileSecondaries(testGuild, map[string]bool{"100": true}, map[string]bool{"100": true}, inFuture(), failOnDelete(t))

	if !hasSecondary(t, "100") {
		t.Error("record of an occupied channel should be kept")
	}
}

func TestReconcile_RecentRecordIsKept(t *testing.T) {
	setupTestDB(t)
	seedSecondary(t, "100")

	reconcileSecondaries(testGuild, map[string]bool{}, map[string]bool{}, time.Now().Add(-time.Minute), failOnDelete(t))

	if !hasSecondary(t, "100") {
		t.Error("record created after the snapshot should be kept")
	}
}

func TestReconcile_OtherGuildIsIgnored(t *testing.T) {
	setupTestDB(t)
	seedSecondary(t, "100")

	reconcileSecondaries("guild-2", map[string]bool{}, map[string]bool{}, inFuture(), failOnDelete(t))

	if !hasSecondary(t, "100") {
		t.Error("record of another guild should be kept")
	}
}

func TestReconcile_DeleteFailureKeepsRecord(t *testing.T) {
	setupTestDB(t)
	seedSecondary(t, "100")

	reconcileSecondaries(testGuild, map[string]bool{"100": true}, map[string]bool{}, inFuture(), func(string) error {
		return errors.New("discord is down")
	})

	if !hasSecondary(t, "100") {
		t.Error("record should survive a failed delete so the next reconciliation retries")
	}
}

func TestReconcile_UnknownChannelDropsRecord(t *testing.T) {
	setupTestDB(t)
	seedSecondary(t, "100")

	reconcileSecondaries(testGuild, map[string]bool{"100": true}, map[string]bool{}, inFuture(), func(string) error {
		return &discordgo.RESTError{Message: &discordgo.APIErrorMessage{Code: discordgo.ErrCodeUnknownChannel}}
	})

	if hasSecondary(t, "100") {
		t.Error("record should be dropped when Discord reports the channel as unknown")
	}
}

func TestGuildCreate_UnavailableGuildIsIgnored(t *testing.T) {
	setupTestDB(t)
	seedSecondary(t, "100")

	GuildCreate(nil, &discordgo.GuildCreate{Guild: &discordgo.Guild{ID: testGuild, Unavailable: true}})

	if !hasSecondary(t, "100") {
		t.Error("an unavailable guild lists no channels and should not drop records")
	}
}

func TestChannelDelete_DropsRecord(t *testing.T) {
	setupTestDB(t)
	seedSecondary(t, "100")
	seedSecondary(t, "200")

	ChannelDelete(nil, &discordgo.ChannelDelete{Channel: &discordgo.Channel{ID: "100"}})

	if hasSecondary(t, "100") {
		t.Error("record of a deleted channel should be dropped")
	}
	if !hasSecondary(t, "200") {
		t.Error("record of another channel should be kept")
	}
}

func TestLiveChannels(t *testing.T) {
	existing, occupied := liveChannels(&discordgo.Guild{
		Channels:    []*discordgo.Channel{{ID: "100"}, {ID: "200"}},
		VoiceStates: []*discordgo.VoiceState{{ChannelID: "200", UserID: "u"}},
	})

	if !existing["100"] || !existing["200"] || len(existing) != 2 {
		t.Errorf("existing = %v", existing)
	}
	if !occupied["200"] || len(occupied) != 1 {
		t.Errorf("occupied = %v", occupied)
	}
}

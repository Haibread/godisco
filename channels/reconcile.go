package channels

import (
	"errors"
	"time"

	"github.com/Haibread/godisco/database"
	"github.com/Haibread/godisco/models"
	"github.com/bwmarrin/discordgo"
)

// liveChannels returns the IDs of the guild's channels and of those holding at
// least one member.
func liveChannels(g *discordgo.Guild) (existing, occupied map[string]bool) {
	existing = make(map[string]bool, len(g.Channels))
	for _, c := range g.Channels {
		existing[c.ID] = true
	}
	occupied = make(map[string]bool, len(g.VoiceStates))
	for _, vs := range g.VoiceStates {
		occupied[vs.ChannelID] = true
	}
	return existing, occupied
}

// reconcileSecondaries repairs the records of a guild against what Discord
// reports. A record whose channel is gone is dropped, and an empty channel is
// deleted along with its record.
//
// Records created at or after before are skipped: their channel may be missing
// from a snapshot taken before they existed, or be waiting for its creator to
// be moved in.
func reconcileSecondaries(guildID string, existing, occupied map[string]bool, before time.Time, deleteChannel func(channelID string) error) {
	var secondaries []models.SecondaryChannel
	if err := database.DB.Where("guild_id = ?", guildID).Find(&secondaries).Error; err != nil {
		log.Errorw("reconcile: list secondary channels", "guild_id", guildID, "error", err)
		return
	}
	log.Debugw("reconcile: start", "guild_id", guildID, "records", len(secondaries))

	for _, c := range secondaries {
		switch {
		case !c.CreatedAt.Before(before):
			log.Debugw("reconcile: skip recent record", "channel_id", c.ChannelID)
			continue
		case occupied[c.ChannelID]:
			log.Debugw("reconcile: channel in use", "channel_id", c.ChannelID)
			continue
		case existing[c.ChannelID]:
			if err := deleteChannel(c.ChannelID); err != nil && !isUnknownChannel(err) {
				log.Errorw("reconcile: delete empty secondary",
					"channel_id", c.ChannelID, "guild_id", guildID, "error", err)
				continue
			}
		}

		log.Infow("reconcile: removing stale secondary", "channel_id", c.ChannelID, "guild_id", guildID, "name", c.Name)
		if err := deleteSecondaryRecord(c.ChannelID); err != nil {
			log.Errorw("reconcile: delete secondary record", "channel_id", c.ChannelID, "error", err)
		}
	}
}

func isUnknownChannel(err error) bool {
	var restErr *discordgo.RESTError
	return errors.As(err, &restErr) && restErr.Message != nil && restErr.Message.Code == discordgo.ErrCodeUnknownChannel
}

func deleteSecondaryRecord(channelID string) error {
	return database.DB.Unscoped().Where("channel_id = ?", channelID).Delete(&models.SecondaryChannel{}).Error
}

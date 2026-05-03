package channels

import (
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/Haibread/godisco/database"
	"github.com/Haibread/godisco/models"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var log *zap.SugaredLogger

// SetLogger injects the logger used by this package. Must be called before
// any handler or loop runs.
func SetLogger(l *zap.SugaredLogger) {
	log = l
}

func userJoined(s *discordgo.Session, i *discordgo.VoiceStateUpdate) {
	channel, err := s.State.Channel(i.ChannelID)
	if err != nil {
		log.Errorw("user joined: lookup channel from state",
			"channel_id", i.ChannelID, "user_id", i.UserID, "error", err)
		return
	}
	if isChannelPrimary(s, i.ChannelID) {
		if _, err := createSecondaryChannelandMove(s, i, channel, i.UserID); err != nil {
			log.Errorw("user joined: create secondary and move",
				"primary_channel_id", channel.ID, "user_id", i.UserID, "error", err)
		}
	}
}

func createSecondaryChannelandMove(s *discordgo.Session, i *discordgo.VoiceStateUpdate, parentChannel *discordgo.Channel, UserID string) (*discordgo.Channel, error) {

	channel, err := s.State.Channel(parentChannel.ID)
	if err != nil {
		return nil, fmt.Errorf("lookup parent channel %s from state: %w", parentChannel.ID, err)
	}

	createdChannel, err := createSecondaryChannel(s, i, channel)
	if err != nil {
		return nil, fmt.Errorf("create secondary under %s: %w", parentChannel.ID, err)
	}

	if err := s.GuildMemberMove(parentChannel.GuildID, UserID, &createdChannel.ID); err != nil {
		return nil, fmt.Errorf("move user %s to %s: %w", UserID, createdChannel.ID, err)
	}

	return createdChannel, nil
}

func createSecondaryChannel(s *discordgo.Session, i *discordgo.VoiceStateUpdate, parentChannel *discordgo.Channel) (*discordgo.Channel, error) {

	channelName, err := getChannelName(s, parentChannel, nil, i.UserID)
	if err != nil {
		return nil, fmt.Errorf("compute channel name: %w", err)
	}

	channelToCreate := &discordgo.GuildChannelCreateData{
		Name:                 channelName,
		Type:                 discordgo.ChannelTypeGuildVoice,
		Bitrate:              parentChannel.Bitrate,
		Position:             parentChannel.Position - 1,
		ParentID:             parentChannel.ParentID,
		PermissionOverwrites: parentChannel.PermissionOverwrites,
	}

	chanCreated, err := s.GuildChannelCreateComplex(parentChannel.GuildID, *channelToCreate)
	if err != nil {
		return nil, fmt.Errorf("discord create channel in guild %s: %w", parentChannel.GuildID, err)
	}

	if err := database.DB.Create(&models.SecondaryChannel{Name: chanCreated.Name, ChannelID: chanCreated.ID, GuildID: chanCreated.GuildID, ParentChannelID: parentChannel.ID, CreatorID: i.UserID}).Error; err != nil {
		return nil, fmt.Errorf("persist secondary channel %s: %w", chanCreated.ID, err)
	}
	return chanCreated, nil
}

func userMoved(s *discordgo.Session, i *discordgo.VoiceStateUpdate) {
	// Check if the channel is managed
	if isChannelPrimary(s, i.BeforeUpdate.ChannelID) {
		log.Debugf("Last channel %v was a primary channel, no action required", i.BeforeUpdate.ChannelID)
	} else if isChannelPrimary(s, i.ChannelID) {
		log.Debugf("Current channel %v is a primary channel, we need to create a new channel", i.ChannelID)
		channel, err := s.Channel(i.ChannelID)
		if err != nil {
			log.Errorw("user moved: fetch primary channel",
				"channel_id", i.ChannelID, "user_id", i.UserID, "error", err)
			return
		}
		if _, err := createSecondaryChannelandMove(s, i, channel, i.UserID); err != nil {
			log.Errorw("user moved: create secondary and move",
				"primary_channel_id", channel.ID, "user_id", i.UserID, "error", err)
		}
	}

	// Check if the channel is in managed channel created
	if isChannelSecondary(s, i.BeforeUpdate.ChannelID) {
		log.Debugf("Last channel %v was a secondary channel, checking if empty", i.BeforeUpdate.ChannelID)
		if isChannelEmpty(s, i.GuildID, i.BeforeUpdate.ChannelID) {
			log.Debugf("Secondary channel %v is empty on guild %v, deleting it", i.BeforeUpdate.ChannelID, i.GuildID)
			if _, err := s.ChannelDelete(i.BeforeUpdate.ChannelID); err != nil {
				log.Errorw("user moved: delete empty secondary",
					"channel_id", i.BeforeUpdate.ChannelID, "guild_id", i.GuildID, "error", err)
			}
			log.Debugf("Removing secondary channel %v record from database.DB", i.BeforeUpdate.ChannelID)
			if err := database.DB.Unscoped().Where("channel_id = ?", i.BeforeUpdate.ChannelID).Delete(&models.SecondaryChannel{}).Error; err != nil {
				log.Errorw("user moved: delete secondary record",
					"channel_id", i.BeforeUpdate.ChannelID, "error", err)
			}
			clearRenameThrottle(i.BeforeUpdate.ChannelID)
		} else {
			log.Debugf("Secondary channel %v is not empty, no actions required", i.BeforeUpdate.ChannelID)
		}
	} else if isChannelSecondary(s, i.ChannelID) {
		log.Debugf("Current channel %v is a secondary channel, no actions required", i.ChannelID)
	}
}

func isChannelEmpty(s *discordgo.Session, GuildID string, ChannelID string) bool {
	guild, err := s.State.Guild(GuildID)
	if err != nil {
		log.Errorw("isChannelEmpty: fetch guild",
			"guild_id", GuildID, "channel_id", ChannelID, "error", err)
		return false
	}

	for _, vs := range guild.VoiceStates {
		if vs.ChannelID == ChannelID {
			return false
		}
	}
	return true
}

func isChannelPrimary(s *discordgo.Session, ChannelID string) bool {
	var channel models.PrimaryChannel
	q := database.DB.Select("channel_id").Where("channel_id = ?", ChannelID).First(&channel)
	if q.Error != nil {
		if !errors.Is(q.Error, gorm.ErrRecordNotFound) {
			log.Error(q.Error)
		}
		return false
	}
	return channel.ChannelID != ""
}

func isChannelSecondary(s *discordgo.Session, ChannelID string) bool {
	var channel models.SecondaryChannel
	q := database.DB.Select("channel_id").Where("channel_id = ?", ChannelID).First(&channel)
	if q.Error != nil {
		if !errors.Is(q.Error, gorm.ErrRecordNotFound) {
			log.Error(q.Error)
		}
		return false
	}
	return channel.ChannelID != ""
}

func getPrimaryChannelTemplate(s *discordgo.Session, ChannelID string) (string, error) {
	var channel models.PrimaryChannel
	managed_channel := database.DB.Select("name_template").Where("channel_id = ?", ChannelID).First(&channel)

	if managed_channel.Error != nil {
		if errors.Is(managed_channel.Error, gorm.ErrRecordNotFound) {
			return "", nil
		} else {
			return "", fmt.Errorf("error while getting channel template: %w", managed_channel.Error)
		}
	}

	if channel.NameTemplate != "" {
		return channel.NameTemplate, nil
	} else {
		return "", fmt.Errorf("no template found for channel %v", ChannelID)
	}
}

func getPrimaryChannelDefaultName(s *discordgo.Session, ChannelID string) (string, error) {
	var channel models.PrimaryChannel
	query := database.DB.Select("name_default").Where("channel_id = ?", ChannelID).First(&channel)

	if query.Error != nil {
		if errors.Is(query.Error, gorm.ErrRecordNotFound) {
			return "", nil
		} else {
			return "", fmt.Errorf("error while getting channel default name: %w", query.Error)
		}
	}

	if channel.NameDefault != "" {
		return channel.NameDefault, nil
	} else {
		return "", fmt.Errorf("no default name found for channel %v", ChannelID)
	}
}

func getSecondaryChannelRank(s *discordgo.Session, ParentChannelID string, ChannelID string) (int, error) {
	// Get all secondary channels for the parent
	var channels []models.SecondaryChannel
	secondary_channels := database.DB.Select("channel_id").Where("parent_channel_id = ?", ParentChannelID).Find(&channels)
	if secondary_channels.Error != nil {
		if errors.Is(secondary_channels.Error, gorm.ErrRecordNotFound) {
			return 1, nil
		} else {
			return 0, fmt.Errorf("error while getting secondary channel count : %w", secondary_channels.Error)
		}
	}

	// Get all the channel_ids
	var secondary_channel_ids []int
	for _, channel := range channels {
		int_channel_id, err := strconv.Atoi(channel.ChannelID)
		if err != nil {
			return 0, fmt.Errorf("error while converting channel ID to int: %w", err)
		}
		secondary_channel_ids = append(secondary_channel_ids, int_channel_id)
	}

	// Sort the channel id
	sort.Ints(secondary_channel_ids)

	// Count and compare
	count := 0
	for _, channel := range secondary_channel_ids {
		int_channel_id := 0
		if ChannelID != "" {
			var err error
			int_channel_id, err = strconv.Atoi(ChannelID)
			if err != nil {
				return 0, fmt.Errorf("error while converting channel ID to int: %w", err)
			}
		}

		if channel == int_channel_id {
			return count + 1, nil
		}
		count += 1
	}

	return count + 1, nil
}

func getChannelName(s *discordgo.Session, parentChannel *discordgo.Channel, secondaryChannel *discordgo.Channel, CreatorID string) (string, error) {
	// Get channel rank
	var channelrank int
	var err error
	if secondaryChannel == nil {
		channelrank, err = getSecondaryChannelRank(s, parentChannel.ID, "")
	} else {
		channelrank, err = getSecondaryChannelRank(s, parentChannel.ID, secondaryChannel.ID)
	}
	if err != nil {
		return "nil", err
	}
	// Get Template
	channelTemplateName, err := getPrimaryChannelTemplate(s, parentChannel.ID)
	if err != nil {
		channelTemplateName = ""

	}

	var channel_tpl = &ChanneltoRename{}
	if secondaryChannel == nil {
		channel_tpl = &ChanneltoRename{
			PrimaryChannel: parentChannel,
			Creator:        CreatorID,
			Template:       channelTemplateName,
			Session:        s,
			Rank:           channelrank,
		}
	} else if secondaryChannel != nil {
		channel_tpl = &ChanneltoRename{
			SecondaryChannel: secondaryChannel,
			Creator:          CreatorID,
			Template:         channelTemplateName,
			Session:          s,
			Rank:             channelrank,
		}
	} else {
		return "nil", fmt.Errorf("error while getting channel type: %w", err)
	}

	var channelName string
	// Get Name from template
	if channel_tpl.Template != "" {
		channelName, err = channel_tpl.getNamefromTemplate()
		if err != nil {
			return "nil", err
		}
	}

	if channelName == "" {
		channelDefaultName, err := getPrimaryChannelDefaultName(s, parentChannel.ID)
		if err != nil {
			return "nil", err
		}
		channelName = fmt.Sprintf("#%d %s", (channelrank)+1, channelDefaultName)
	}
	return channelName, nil
}

// PrimarySummary is the read-model returned to /list-primaries.
type PrimarySummary struct {
	ChannelID      string
	DefaultName    string
	Template       string
	SecondaryCount int
}

// ListPrimarySummaries returns every primary channel registered for the
// given guild, with a count of currently-spawned secondaries for each.
func ListPrimarySummaries(guildID string) ([]PrimarySummary, error) {
	var primaries []models.PrimaryChannel
	if err := database.DB.Where("guild_id = ?", guildID).Find(&primaries).Error; err != nil {
		return nil, fmt.Errorf("list primary channels: %w", err)
	}
	out := make([]PrimarySummary, 0, len(primaries))
	for _, p := range primaries {
		var count int64
		if err := database.DB.Model(&models.SecondaryChannel{}).Where("parent_channel_id = ?", p.ChannelID).Count(&count).Error; err != nil {
			return nil, fmt.Errorf("count secondaries for %s: %w", p.ChannelID, err)
		}
		out = append(out, PrimarySummary{
			ChannelID:      p.ChannelID,
			DefaultName:    p.NameDefault,
			Template:       p.NameTemplate,
			SecondaryCount: int(count),
		})
	}
	return out, nil
}

// DeletePrimaryChannel removes a primary channel from Discord and the DB.
// It refuses to delete primaries that still have spawned secondaries so we
// don't orphan live voice channels.
func DeletePrimaryChannel(s *discordgo.Session, guildID, channelID string) error {
	var primary models.PrimaryChannel
	q := database.DB.Where("channel_id = ? AND guild_id = ?", channelID, guildID).First(&primary)
	if errors.Is(q.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("channel %s is not a managed primary in this server", channelID)
	}
	if q.Error != nil {
		return fmt.Errorf("lookup primary channel: %w", q.Error)
	}

	var secondaryCount int64
	if err := database.DB.Model(&models.SecondaryChannel{}).Where("parent_channel_id = ?", channelID).Count(&secondaryCount).Error; err != nil {
		return fmt.Errorf("count secondary channels: %w", err)
	}
	if secondaryCount > 0 {
		return fmt.Errorf("primary still has %d active secondary channel(s); wait for them to empty before deleting", secondaryCount)
	}

	if _, err := s.ChannelDelete(channelID); err != nil {
		return fmt.Errorf("delete discord channel: %w", err)
	}
	if err := database.DB.Unscoped().Where("channel_id = ?", channelID).Delete(&models.PrimaryChannel{}).Error; err != nil {
		return fmt.Errorf("delete primary channel record: %w", err)
	}
	return nil
}

func CreatePrimaryChannel(s *discordgo.Session, GuildID string, NameTemplate string, NameDefault string) (*discordgo.Channel, error) {

	channelToCreate := &discordgo.GuildChannelCreateData{
		Name: "➕ New Channel",
		Type: discordgo.ChannelTypeGuildVoice,
	}

	// Create the new channel
	chanCreated, err := s.GuildChannelCreateComplex(GuildID, *channelToCreate)
	if err != nil {
		return nil, err
	}

	// Add channel in database.database.DB
	query := database.DB.Create(&models.PrimaryChannel{NameTemplate: NameTemplate, NameDefault: NameDefault, ChannelID: chanCreated.ID, GuildID: GuildID})

	if query.Error != nil {
		return nil, query.Error
	}

	return chanCreated, nil
}

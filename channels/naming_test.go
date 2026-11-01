package channels

import (
	"testing"

	"github.com/Haibread/godisco/database"
	"github.com/Haibread/godisco/models"
	"github.com/bwmarrin/discordgo"
)

func TestGetChannelName_DefaultNameStartsAtOne(t *testing.T) {
	setupTestDB(t)
	if err := database.DB.Create(&models.PrimaryChannel{ChannelID: testParent, NameDefault: "General"}).Error; err != nil {
		t.Fatalf("seed primary: %v", err)
	}

	name, err := getChannelName(nil, &discordgo.Channel{ID: testParent}, nil, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "#1 General" {
		t.Errorf("name = %q, want %q", name, "#1 General")
	}
}

func TestGetNamefromTemplate_FirstRank(t *testing.T) {
	setupTestDB(t)

	cases := map[string]string{
		"{{.Icao}}":    "Alfa",
		"#{{.Number}}": "#1",
	}
	for tpl, want := range cases {
		got, err := ChanneltoRename{Rank: 1, Template: tpl}.getNamefromTemplate()
		if err != nil {
			t.Fatalf("template %q: %v", tpl, err)
		}
		if got != want {
			t.Errorf("template %q = %q, want %q", tpl, got, want)
		}
	}
}

func TestGetICAO_WrapsPastTheAlphabet(t *testing.T) {
	if got := getICAO(len(icao)); got != "Alfa" {
		t.Errorf("getICAO(%d) = %q, want %q", len(icao), got, "Alfa")
	}
}

func sessionWithGuild(t *testing.T, g *discordgo.Guild) *discordgo.Session {
	t.Helper()
	s := &discordgo.Session{State: discordgo.NewState()}
	if err := s.State.GuildAdd(g); err != nil {
		t.Fatalf("add guild: %v", err)
	}
	return s
}

func TestGetMainActivity_MemberWithoutPresence(t *testing.T) {
	setupTestDB(t)
	s := sessionWithGuild(t, &discordgo.Guild{
		ID: testGuild,
		VoiceStates: []*discordgo.VoiceState{
			{UserID: "no-presence", ChannelID: "100"},
			{UserID: "playing", ChannelID: "100"},
		},
		Presences: []*discordgo.Presence{{
			User:       &discordgo.User{ID: "playing"},
			Activities: []*discordgo.Activity{{Name: "Aniimo", Type: discordgo.ActivityTypeGame}},
		}},
	})

	got, err := getMainActivity(s, &discordgo.Channel{ID: "100", GuildID: testGuild})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Aniimo" {
		t.Errorf("activity = %q, want %q", got, "Aniimo")
	}
}

func TestGetMainActivityUser_UserWithoutPresence(t *testing.T) {
	setupTestDB(t)
	s := sessionWithGuild(t, &discordgo.Guild{ID: testGuild})

	got, err := getMainActivityUser(s, &discordgo.Channel{GuildID: testGuild}, &discordgo.User{ID: "no-presence"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("activity = %q, want empty", got)
	}
}

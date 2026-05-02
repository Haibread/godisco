package commands

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestOptionString(t *testing.T) {
	options := []*discordgo.ApplicationCommandInteractionDataOption{
		{Name: "default-name", Value: "General"},
		{Name: "template", Value: "{{.Icao}}"},
		{Name: "count", Value: float64(3)},
	}

	tests := []struct {
		name      string
		key       string
		wantValue string
		wantOk    bool
	}{
		{"existing string option", "default-name", "General", true},
		{"second string option", "template", "{{.Icao}}", true},
		{"missing option", "nope", "", false},
		{"non-string option", "count", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := optionString(options, tt.key)
			if got != tt.wantValue || ok != tt.wantOk {
				t.Errorf("optionString(%q) = (%q, %v), want (%q, %v)", tt.key, got, ok, tt.wantValue, tt.wantOk)
			}
		})
	}
}

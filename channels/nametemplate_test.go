package channels

import (
	"reflect"
	"testing"
)

func TestNeededVariables(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     []string
	}{
		{"empty", "", []string{}},
		{"no variables", "Static Name", []string{}},
		{"single variable", "{{.Icao}}", []string{"Icao"}},
		{"multiple variables", "{{.Icao}} {{.GameName}}", []string{"Icao", "GameName"}},
		{"variables with whitespace", "{{ .Icao }}-{{ .Number }}", []string{"Icao", "Number"}},
		{"duplicate variables", "{{.GameName}}/{{.GameName}}", []string{"GameName", "GameName"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := neededVariables(tt.template)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("neededVariables(%q) = %v, want %v", tt.template, got, tt.want)
			}
		})
	}
}

func TestGetICAO(t *testing.T) {
	cases := map[int]string{
		0:  "Alfa",
		1:  "Beta",
		7:  "Hotel",
		25: "Zulu",
	}
	for idx, want := range cases {
		if got := getICAO(idx); got != want {
			t.Errorf("getICAO(%d) = %q, want %q", idx, got, want)
		}
	}
}

func TestTestTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{"empty template", "", false},
		{"static text", "Channel", false},
		{"valid variable", "{{.Icao}} {{.GameName}}", false},
		{"unclosed action", "{{.Icao", true},
		{"invalid syntax", "{{ .Icao }} {{ end", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := TestTemplate(nil, tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestTemplate(%q) err = %v, wantErr %v", tt.template, err, tt.wantErr)
			}
		})
	}
}

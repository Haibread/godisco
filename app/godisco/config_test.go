package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// newViper mirrors the setup initconfig does on the global viper.
func newViper() *viper.Viper {
	v := viper.New()
	v.SetDefault("token", "")
	v.SetDefault("bot_status", "")
	v.SetDefault("bot_activity_type", "Listening")
	v.SetConfigName("config")
	v.AddConfigPath("config")
	v.SetConfigType("yaml")
	return v
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		env       map[string]string
		args      []string
		wantToken string
		wantDB    string
		wantErr   bool
	}{
		{
			name:      "defaults read config/config.yaml and keep the database beside it",
			files:     map[string]string{"config/config.yaml": "token: file"},
			wantToken: "file",
			wantDB:    filepath.Join("config", "channels.db"),
		},
		{
			name:      "db_path in the config file overrides the default",
			files:     map[string]string{"config/config.yaml": "token: file\ndb_path: from-file.db"},
			wantToken: "file",
			wantDB:    "from-file.db",
		},
		{
			name:      "DB_PATH beats the config file",
			files:     map[string]string{"config/config.yaml": "token: file\ndb_path: from-file.db"},
			env:       map[string]string{"DB_PATH": "from-env.db"},
			wantToken: "file",
			wantDB:    "from-env.db",
		},
		{
			name:      "--db-path beats DB_PATH",
			files:     map[string]string{"config/config.yaml": "token: file"},
			env:       map[string]string{"DB_PATH": "from-env.db"},
			args:      []string{"--db-path", "from-flag.db"},
			wantToken: "file",
			wantDB:    "from-flag.db",
		},
		{
			name: "CONFIG_FILE replaces the search in ./config",
			files: map[string]string{
				"config/config.yaml": "token: default",
				"elsewhere/bot.yaml": "token: moved",
			},
			env:       map[string]string{"CONFIG_FILE": "elsewhere/bot.yaml"},
			wantToken: "moved",
			wantDB:    filepath.Join("config", "channels.db"),
		},
		{
			name: "--config-file beats CONFIG_FILE",
			files: map[string]string{
				"a.yaml": "token: env",
				"b.yaml": "token: flag",
			},
			env:       map[string]string{"CONFIG_FILE": "a.yaml"},
			args:      []string{"--config-file", "b.yaml"},
			wantToken: "flag",
			wantDB:    filepath.Join("config", "channels.db"),
		},
		{
			name:      "DISCORD_TOKEN beats the config file",
			files:     map[string]string{"config/config.yaml": "token: file"},
			env:       map[string]string{"DISCORD_TOKEN": "env"},
			wantToken: "env",
			wantDB:    filepath.Join("config", "channels.db"),
		},
		{
			name:    "a missing default config file is fatal without DISCORD_TOKEN",
			wantErr: true,
		},
		{
			name:      "a missing default config file is fine with DISCORD_TOKEN",
			env:       map[string]string{"DISCORD_TOKEN": "env"},
			wantToken: "env",
			wantDB:    filepath.Join("config", "channels.db"),
		},
		{
			name:    "a missing explicit config file is fatal even with DISCORD_TOKEN",
			env:     map[string]string{"DISCORD_TOKEN": "env", "CONFIG_FILE": "absent.yaml"},
			wantErr: true,
		},
		{
			name:    "an empty db_path is rejected",
			files:   map[string]string{"config/config.yaml": "token: file\ndb_path: \"\""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			for _, env := range envNames {
				t.Setenv(env, "")
			}
			for k, val := range tt.env {
				t.Setenv(k, val)
			}
			for path, content := range tt.files {
				writeFile(t, path, content)
			}

			v := newViper()
			err := loadConfig(v, tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("want an error, got none")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := v.GetString(keyToken); got != tt.wantToken {
				t.Errorf("token = %q, want %q", got, tt.wantToken)
			}
			if got := v.GetString(keyDBPath); got != tt.wantDB {
				t.Errorf("db_path = %q, want %q", got, tt.wantDB)
			}
		})
	}
}

func TestLoadConfigActivityType(t *testing.T) {
	tests := []struct {
		name string
		file string
		env  string
		want string
	}{
		{name: "defaults to Listening", file: "token: file", want: "Listening"},
		{name: "the config file overrides the default", file: "token: file\nbot_activity_type: Watching", want: "Watching"},
		{name: "BOT_ACTIVITY_TYPE beats the config file", file: "token: file\nbot_activity_type: Watching", env: "Playing", want: "Playing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			for _, env := range envNames {
				t.Setenv(env, "")
			}
			t.Setenv("BOT_ACTIVITY_TYPE", tt.env)
			writeFile(t, "config/config.yaml", tt.file)

			v := newViper()
			if err := loadConfig(v, nil); err != nil {
				t.Fatal(err)
			}
			if got := v.GetString("bot_activity_type"); got != tt.want {
				t.Errorf("bot_activity_type = %q, want %q", got, tt.want)
			}
		})
	}
}

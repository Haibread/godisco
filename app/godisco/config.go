package main

import (
	"errors"
	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	keyConfigFile = "config_file"
	keyDBPath     = "db_path"
	keyToken      = "token"
)

var envNames = map[string]string{
	keyConfigFile:       "CONFIG_FILE",
	keyDBPath:           "DB_PATH",
	keyToken:            "DISCORD_TOKEN",
	"bot_status":        "BOT_STATUS",
	"bot_activity_type": "BOT_ACTIVITY_TYPE",
}

// loadConfig layers command-line flags and environment variables over the
// config file and the defaults already set on v, then reads the config file.
func loadConfig(v *viper.Viper, args []string) error {
	flags := pflag.NewFlagSet("godisco", pflag.ExitOnError)
	flags.String("config-file", "", "config file to read instead of searching ./config for config.yaml")
	flags.String("db-path", "", "SQLite database file (default \"config/channels.db\")")
	if err := flags.Parse(args); err != nil {
		return err
	}

	v.SetDefault(keyDBPath, filepath.Join("config", "channels.db"))
	for key, env := range envNames {
		if err := v.BindEnv(key, env); err != nil {
			return err
		}
	}
	if err := v.BindPFlag(keyConfigFile, flags.Lookup("config-file")); err != nil {
		return err
	}
	if err := v.BindPFlag(keyDBPath, flags.Lookup("db-path")); err != nil {
		return err
	}

	if err := readConfigFile(v); err != nil {
		return err
	}
	if v.GetString(keyDBPath) == "" {
		return errors.New("db_path is empty")
	}
	return nil
}

// readConfigFile reads the file named by config_file, or searches for the
// default one. Only the default may be absent, and only when the token comes
// from elsewhere.
func readConfigFile(v *viper.Viper) error {
	if path := v.GetString(keyConfigFile); path != "" {
		v.SetConfigFile(path)
		return v.ReadInConfig()
	}
	err := v.ReadInConfig()
	if notFound := (viper.ConfigFileNotFoundError{}); errors.As(err, &notFound) && v.GetString(keyToken) != "" {
		return nil
	}
	return err
}

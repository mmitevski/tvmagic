package common

import (
	"github.com/mmitevski/transactions/db"
	"flag"
	"gopkg.in/gcfg.v1"
	"log"
	"os"
)

type UI struct {
	IntroSubTitle string
}

type ServerConfig struct {
	Address string
}

type SessionConfig struct {
	Cookie      string
	MaxLifeTime int64
	Secure      bool
}

type AuthenticationConfig struct {
	Command string
}

type Config struct {
	Database       db.DatabaseConfig
	Server         ServerConfig
	Session        SessionConfig
	Authentication AuthenticationConfig
	UI             UI
}

var (
	configFile = "tvmagic.ini"
	config *Config
)

func init() {
	flag.StringVar(&configFile, "config", "tvmagic.ini", "Configuration file for the application")
	flag.Parse()
}

func GetConfig() *Config {
	if config == nil {
		var c Config
		c.Session.Cookie = "session"
		c.Session.MaxLifeTime = 3600
		c.Session.Secure = false
		err := gcfg.ReadFileInto(&c, configFile)
		if err != nil {
			log.Printf("Failed to parse configuration file %s: %v", configFile, err)
			os.Exit(1)
		}
		config = &c
	}
	return config
}
package config

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"log"
	"os"
	"time"

	"github.com/spf13/viper"
)

//go:embed config-default.yml
var defaultConfig embed.FS

type RedisConfig struct {
	Host               string
	Port               string
	Password           string
	Db                 string
	DialTimeout        time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleCheckFrequency time.Duration
	PoolSize           int
	PoolTimeout        time.Duration
}

type ServerConfig struct {
	ServerPort          uint16
	InternalPort        string
	ExternalPort        string
	RunMode             string
	TimeZone            string
	Secret              string
	SessionTimeoutHours int
}

type Config struct {
	Server ServerConfig
	Redis  RedisConfig
}

func getConfigPath(env string) string {
	if env == "docker" {
		return "/app/config/config-docker"
	} else if env == "production" {
		return "/config/config-production"
	} else {
		return "backend_golang/config/config-default.yml"
	}
}

func GetConfig() *Config {
	cfgPath := getConfigPath(os.Getenv("APP_ENV"))
	cfg, err := GetConfigFromPath((cfgPath))
	if err != nil {
		log.Fatalf("Error in parse config %v", err)
	}
	return cfg
}

func ParseConfig(v *viper.Viper) (*Config, error) {
	var cfg Config
	err := v.Unmarshal(&cfg)
	if err != nil {
		log.Printf("Unable to parse config: %v", err)
		return nil, err
	}
	return &cfg, nil
}
func LoadConfig(filename string, fileType string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigType(fileType)
	v.SetConfigName(filename)
	v.AddConfigPath(".")
	v.AutomaticEnv()

	err := v.ReadInConfig()
	if err == nil {
		return v, nil
	}
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		// load in from default

		cached, err := fs.ReadFile(defaultConfig, "config-default.yml")
		if err != nil {
			return nil, errors.New("config file not found")
		}
		v.ReadConfig(bytes.NewBuffer(cached))
		return v, nil
	}

	log.Printf("Unable to read config: %v", err)
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		return nil, errors.New("config file not found")
	}
	return nil, err

}

func GetConfigFromPath(cfgPath string) (*Config, error) {
	v, err := LoadConfig(cfgPath, "yml")
	if err != nil {
		log.Fatalf("Error in load config %v", err)
	}

	cfg, err := ParseConfig(v)
	envPort := os.Getenv("PORT")
	if envPort != "" {
		cfg.Server.ExternalPort = envPort
		log.Printf("Set external port from environment -> %s", cfg.Server.ExternalPort)
	} else {
		cfg.Server.ExternalPort = cfg.Server.InternalPort
		log.Printf("Set external port from environment -> %s", cfg.Server.ExternalPort)
	}
	if err != nil {
		log.Fatalf("Error in parse config %v", err)
	}
	return cfg, err
}

package config

import (
	"embed"
	"errors"
	"log"
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

func LoadConfig(filename string, fileType string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigType(fileType)
	v.SetConfigName(filename)
	v.AddConfigPath(".")
	v.AutomaticEnv()

	err := v.ReadInConfig()
	if err != nil {
		log.Printf("Unable to read config, : %v Using default options", err)
		
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {

			return nil, errors.New("config file not found")
		}
		return nil, err
	}
	return v, nil
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

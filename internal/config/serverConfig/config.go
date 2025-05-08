package serverConfig

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"log/slog"
	"os"
	"time"
)

const (
	CONFIG_SERVER_PATH = "CONFIG_SERVER_PATH"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	Kafka  KafkaConfig  `yaml:"kafka"`
}

type ServerConfig struct {
	Address string        `yaml:"address" env-default:"localhost:50051"`
	TimeOut time.Duration `yaml:"timeout" env-default:"8s"`
	Type    string        `yaml:"type" env-default:"tcp"`
}

type KafkaConfig struct {
	Broker          string `yaml:"broker" env-default:"localhost:9092"`
	InvitationTopic string `yaml:"invitation_topic" env-default:"invitation_topic"`
}

func MustLoadServerConfig() (*Config, error) {

	slog.Debug("Loading server config")

	configPath := os.Getenv(CONFIG_SERVER_PATH)
	if configPath == "" {
		return nil, fmt.Errorf("%s environment variable not set", CONFIG_SERVER_PATH)
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%s does not exist %s", CONFIG_SERVER_PATH, configPath)
	}

	var config Config

	if err := cleanenv.ReadConfig(configPath, &config); err != nil {
		return nil, fmt.Errorf("cannot load config file: %s", err)
	}

	return &config, nil
}

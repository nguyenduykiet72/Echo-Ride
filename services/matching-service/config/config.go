package config

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Redis        RedisConfig        `yaml:"redis"`
	Kafka        KafkaConfig        `yaml:"kafka"`
	Dependencies DependenciesConfig `yaml:"dependencies"`
}

type ServerConfig struct {
	Port string `yaml:"port" env:"MATCHING_PORT" env-default:"8113" validate:"required"`
	Mode string `yaml:"mode" env:"MATCHING_MODE" env-default:"development" validate:"required"`
}

type RedisConfig struct {
	Host     string `yaml:"host" env:"MATCHING_REDIS_HOST" validate:"required"`
	Port     int    `yaml:"port" env:"MATCHING_REDIS_PORT" validate:"required"`
	Password string `yaml:"password" env:"MATCHING_REDIS_PASSWORD"`
	DB       int    `yaml:"db" env:"MATCHING_REDIS_DB" env-default:"0"`
}

type KafkaConfig struct {
	Brokers []string `yaml:"brokers" env:"MATCHING_KAFKA_BROKERS" validate:"required"`
	Topic   string   `yaml:"topic" env:"MATCHING_KAFKA_TOPIC" validate:"required"`
	GroupID string   `yaml:"group_id" env:"MATCHING_KAFKA_GROUP_ID" validate:"required"`
}

type DependenciesConfig struct {
	LocationGrpcUrl string `yaml:"location_grpc_url" env:"LOCATION_GRPC_URL" validate:"required"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}

	configPath := os.Getenv("MATCHING_CONFIG_PATH")
	if configPath == "" {
		configPath = "./config/config.dev.yml" // default fallback
	}

	err := cleanenv.ReadConfig(configPath, cfg)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		if envErr := cleanenv.ReadEnv(cfg); envErr != nil {
			return nil, fmt.Errorf("Error loading config from environment variables: %v\n", envErr)
		}
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("config validation error: %v", err)
	}

	return cfg, nil
}

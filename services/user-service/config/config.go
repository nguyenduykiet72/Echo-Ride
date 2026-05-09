package config

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig `yaml:"server"`
	Database DBConfig     `yaml:"database"`
	Kafka    KafkaConfig  `yaml:"kafka"`
	Jaeger   JaegerConfig `yaml:"jaeger"`
	GRPC     GRPCConfig   `yaml:"grpc"`
}

type ServerConfig struct {
	Port string `yaml:"port" env:"USER_PORT" env-default:"8115" validate:"required"`
	Mode string `yaml:"mode" env:"USER_MODE" env-default:"development" validate:"required"`
}

type GRPCConfig struct {
	Port string `yaml:"port" env:"USER_GRPC_PORT" env-default:"9115" validate:"required"`
}

type DBConfig struct {
	Host     string `yaml:"host" env:"USER_DB_HOST" validate:"required"`
	Port     int    `yaml:"port" env:"USER_DB_PORT" validate:"required"`
	User     string `yaml:"username" env:"USER_DB_USER" validate:"required"`
	Password string `yaml:"password" env:"USER_DB_PASSWORD" validate:"required"`
	DBName   string `yaml:"dbname" env:"USER_DB_NAME" validate:"required"`
	Type     string `yaml:"type" env:"USER_DB_TYPE" env-default:"postgres" validate:"required,oneof=postgres mysql"`
}

type KafkaConfig struct {
	Brokers       []string `yaml:"brokers" env:"USER_KAFKA_BROKERS" validate:"required"`
	Topic         string   `yaml:"topic" env:"USER_KAFKA_TOPIC" validate:"required"`
	IdentityTopic string   `yaml:"identity_topic" env:"USER_KAFKA_IDENTITY_TOPIC" env-default:"auth.events" validate:"required"`
}

type JaegerConfig struct {
	AgentHost string `yaml:"service_host" env:"USER_JAEGER_AGENT_HOST" env-default:"localhost" validate:"required"`
	AgentPort int    `yaml:"service_port" env:"USER_JAEGER_AGENT_PORT" env-default:"4317" validate:"required"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}

	configPath := os.Getenv("USER_CONFIG_PATH")
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

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
	JWT      JWTConfig    `yaml:"jwt"`
}

type ServerConfig struct {
	Port string `yaml:"port" env:"AUTH_PORT" env-default:"8082" validate:"required"`
	Mode string `yaml:"mode" env:"AUTH_MODE" env-default:"development" validate:"required"`
}

type DBConfig struct {
	Host     string `yaml:"host" env:"AUTH_DB_HOST" validate:"required"`
	Port     int    `yaml:"port" env:"AUTH_DB_PORT" validate:"required"`
	User     string `yaml:"username" env:"AUTH_DB_USER" validate:"required"`
	Password string `yaml:"password" env:"AUTH_DB_PASSWORD" validate:"required"`
	DBName   string `yaml:"dbname" env:"AUTH_DB_NAME" validate:"required"`
	Type     string `yaml:"type" env:"AUTH_DB_TYPE" env-default:"postgres" validate:"required,oneof=postgres mysql"`
}

type KafkaConfig struct {
	Brokers []string `yaml:"brokers" env:"AUTH_KAFKA_BROKERS" validate:"required"`
	Topic   string   `yaml:"topic" env:"AUTH_KAFKA_TOPIC" validate:"required"`
}

type JaegerConfig struct {
	AgentHost string `yaml:"service_host" env:"AUTH_JAEGER_AGENT_HOST" env-default:"localhost" validate:"required"`
	AgentPort int    `yaml:"service_port" env:"AUTH_JAEGER_AGENT_PORT" env-default:"4317" validate:"required"`
}

type JWTConfig struct {
	SecretKey      string `yaml:"secret" env:"AUTH_JWT_SECRET" validate:"required"`
	ExpirationTime int    `yaml:"expiration_time" env:"AUTH_JWT_EXPIRATION_TIME" env-default:"1440" validate:"required"` // in minutes
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}

	configPath := os.Getenv("RIDE_CONFIG_PATH")
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

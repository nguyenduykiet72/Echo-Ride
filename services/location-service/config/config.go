package config

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	Redis  RedisConfig  `yaml:"redis"`
	//Kafka  KafkaConfig  `yaml:"kafka"`
	Grpc   GrpcConfig   `yaml:"grpc"`
	Jaeger JaegerConfig `yaml:"jaeger"`
}

type ServerConfig struct {
	Port string `yaml:"port" env:"LOCATION_PORT" env-default:"8080" validate:"required"`
	Mode string `yaml:"mode" env:"LOCATION_MODE" env-default:"development" validate:"required"`
}

type RedisConfig struct {
	Host     string `yaml:"host" env:"LOCATION_REDIS_HOST" validate:"required"`
	Port     int    `yaml:"port" env:"LOCATION_REDIS_PORT" validate:"required"`
	Password string `yaml:"password" env:"LOCATION_REDIS_PASSWORD"`
	DB       int    `yaml:"db" env:"LOCATION_REDIS_DB" env-default:"0"`
}

//type KafkaConfig struct {
//	Brokers []string `yaml:"brokers" env:"LOCATION_KAFKA_BROKERS" validate:"required"`
//	Topic   string   `yaml:"topic" env:"LOCATION_KAFKA_TOPIC" validate:"required"`
//}

type GrpcConfig struct {
	Port string `yaml:"port" env:"LOCATION_GRPC_PORT" env-default:"9090" validate:"required"`
}

type JaegerConfig struct {
	AgentHost string `yaml:"service_host" env:"RIDE_JAEGER_AGENT_HOST" env-default:"localhost" validate:"required"`
	AgentPort int    `yaml:"service_port" env:"RIDE_JAEGER_AGENT_PORT" env-default:"4317" validate:"required"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}

	configPath := os.Getenv("LOCATION_CONFIG_PATH")
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

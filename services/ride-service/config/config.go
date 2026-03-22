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
}

type ServerConfig struct {
	Port string `yaml:"port" env:"RIDE_PORT" env-default:"8080" validate:"required"`
	Mode string `yaml:"mode" env:"RIDE_MODE" env-default:"development" validate:"required"`
}

type DBConfig struct {
	Host     string `yaml:"host" env:"RIDE_DB_HOST" validate:"required"`
	Port     int    `yaml:"port" env:"RIDE_DB_PORT" validate:"required"`
	User     string `yaml:"username" env:"RIDE_DB_USER" validate:"required"`
	Password string `yaml:"password" env:"RIDE_DB_PASSWORD" validate:"required"`
	DBName   string `yaml:"dbname" env:"RIDE_DB_NAME" validate:"required"`
	Type     string `yaml:"type" env:"RIDE_DB_TYPE" env-default:"postgres" validate:"required,oneof=postgres mysql"`
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

package config

import (
	"fmt"
	"github.com/zhukovrost/cadv_email/internal/delivery"
	"github.com/zhukovrost/cadv_email/internal/service"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	RabbitMQ delivery.Config `yaml:"rabbitmq"`
	SMTP     service.Config  `yaml:"smtp"`
}

func New() (*Config, error) {
	var cfg Config

	if err := loadConfig("config/config.yml", &cfg); err != nil {
		return nil, err
	}
	if err := processEnvironment(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadConfig(filename string, cfg *Config) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open config file %s: %w", filename, err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return fmt.Errorf("failed to decode YAML from config file %s: %w", filename, err)
	}

	return nil
}

func processEnvironment(cfg *Config) error {
	if user, exists := os.LookupEnv("SMTP_USER"); exists {
		cfg.SMTP.Username = user
		cfg.SMTP.Sender = user
	}
	if password, exists := os.LookupEnv("SMTP_PASSWORD"); exists {
		cfg.SMTP.Password = password
	}
	return nil
}

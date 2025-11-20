package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	AppName    string         `yaml:"app_name" env:"APP_NAME"`
	Version    string         `yaml:"version"  env:"APP_VERSION"`
	APIS       []API          `yaml:"apis"`
	Components Components     `yaml:"components"`
	AwsConfig  AWSConfig      `yaml:"aws_config"`
	Database   DatabaseConfig `yaml:"database"`
	Auth       AuthConfig     `yaml:"auth"`
	Certs      CertsConfig    `yaml:"certs"`
	Server     ServerConfig   `yaml:"server"`
	Logger     LoggerConfig   `yaml:"logger"`
}

type API struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
	Key  string `yaml:"-"`
}

type Components struct {
	HetznerCli      string `yaml:"hetzner_cli"`
	CloudflareCli   string `yaml:"cloudflare_cli"`
	Route53Cli      string `yaml:"route_53_cli"`
	DigitalOceanCli string `yaml:"digitalocean_cli"`
	AuthCli         string `yaml:"auth_cli"`
}

type AWSConfig struct {
	AccessKey string `yaml:"access_key" env:"AWS_ACCESS_KEY"`
	SecretKey string `yaml:"secret_key" env:"AWS_SECRET_KEY"`
	Region    string `yaml:"region"`
}

type DatabaseConfig struct {
	Name          string `yaml:"name"`
	Host          string `yaml:"host" env:"DB_HOST"`
	Port          int    `yaml:"port" env:"DB_PORT"`
	User          string `yaml:"user" env:"DB_USER"`
	Password      string `yaml:"password" env:"DB_PASSWORD"`
	Database      string `yaml:"database"`
	MigrationPath string `yaml:"migration_path"`
}

type AuthConfig struct {
	AccessSecKey  string `yaml:"access_sec_key"  env:"AUTH_ACCESS_KEY"`
	RefreshSecKey string `yaml:"refresh_sec_key" env:"AUTH_REFRESH_KEY"`
}

type CertsConfig struct {
	StorageDir      string        `yaml:"storage_dir"`
	Email           string        `yaml:"email"`
	RenewalDuration time.Duration `yaml:"renewal_duration" env:"CERT_RENEWAL_DURATION"`
}

type ServerConfig struct {
	Port string `yaml:"port" env:"SERVER_PORT"`
}

type LoggerConfig struct {
	LogLevel string `yaml:"log_level" env:"LOG_LEVEL"`
}

func LoadConfig(confPath string) (*Config, error) {
	if confPath == "" {
		return nil, errors.New("config path is empty")
	}

	if _, err := os.Stat(confPath); err != nil {
		return nil, errors.New("config file does not exist: " + confPath)
	}

	var cfg Config

	// Load YAML file
	if err := cleanenv.ReadConfig(confPath, &cfg); err != nil {
		return nil, err
	}

	// Load Api.Key values
	for i := range cfg.APIS {
		api := &cfg.APIS[i]

		envName := "API_KEY_" + strings.ToUpper(api.Name)

		api.Key = os.Getenv(envName)
		if api.Key == "" {
			return nil, fmt.Errorf("missing environment variable %s for API '%s'", envName, api.Name)
		}
	}

	// Override with environment variables
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

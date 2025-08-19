package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   Server   `yaml:"server"`
	Database Database `yaml:"database"`
	Redis Redis `yaml:"redis"`
}

type Server struct {
	Host string `yaml:"host" env-default:":8080"`
	Port string `yaml:"port" env-default:"localhost"`
}

type Database struct {
	Name     string `yaml:"name" env-default:"postgres"`
	User     string `yaml:"user" env-default:"postgres"`
	Password string `yaml:"password" env-default:"postgres"`
	Host     string `yaml:"host" env-default:"localhost"`
	Port     string `yaml:"port" env-default:"5432"`
}

type Redis struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

func MustLoad() *Config {
	configPath := "config/default.yaml"

	if err := godotenv.Load(); err != nil {
		panic(fmt.Sprintf("ошибка загрузки файла .env: %v", err))
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic(fmt.Sprintf("конфигурационный файл не существует: %s", configPath))
	}

	yamlFile, err := os.ReadFile(configPath)
	if err != nil {
		panic(fmt.Sprintf("не удалось прочитать конфигурационный файл: %v", err))
	}

	var config Config

	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		panic(fmt.Sprintf("не удалось распарсить конфигурационный файл: %v", err))
	}

	return &config
}

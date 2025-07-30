package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
	"os"
)

// Config содержит все конфигурационные параметры приложения
type Config struct {
	Server   Server   `yaml:"server"`   // Настройки HTTP сервера
	Database Database `yaml:"database"` // Настойки базы данных
}

type Server struct {
	Host string `yaml:"host" env-default:":8080"`     // Адрес и порт сервера
	Port string `yaml:"port" env-default:"localhost"` // Таймаут запросов
}

type Database struct {
	Name     string `yaml:"name" env-default:"postgres"`
	User     string `yaml:"user" env-default:"postgres"`
	Password string `yaml:"password" env-default:"postgres"`
	Host     string `yaml:"host" env-default:"localhost"`
	Port     string `yaml:"port" env-default:"5432"`
}

// MustLoad загружает конфигурацию из файла и переменных окружения
// Если загрузка не удалась, приложение завершается с ошибкой
func MustLoad() *Config {
	configPath := "config/default.yaml"

	// Загрузка переменных окружения из .env файла
	if err := godotenv.Load(); err != nil {
		panic(fmt.Sprintf("ошибка загрузки файла .env: %v", err))
	}

	// Проверка существования файла конфигурации
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic(fmt.Sprintf("конфигурационный файл не существует: %s", configPath))
	}

	// Чтение содержимого файла
	yamlFile, err := os.ReadFile(configPath)
	if err != nil {
		panic(fmt.Sprintf("не удалось прочитать конфигурационный файл: %v", err))
	}

	var config Config

	// Разбор YAML-файла
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		panic(fmt.Sprintf("не удалось распарсить конфигурационный файл: %v", err))
	}

	return &config
}

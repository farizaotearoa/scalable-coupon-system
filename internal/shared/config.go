package shared

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	DBUsername string
	DBPassword string
	DBHost     string
	DBPort     int
	DBName     string
	Production bool
	LogPath    string
	AppPort    string
}

func NewConfig() *Config {
	cfg := Config{}
	cfg.DBUsername = os.Getenv("DB_USERNAME")
	cfg.DBPassword = os.Getenv("DB_PASSWORD")
	cfg.DBHost = os.Getenv("DB_HOST")
	cfg.DBPort = cfg.getEnvInt("DB_PORT", 5432)
	cfg.DBName = os.Getenv("DB_NAME")
	cfg.LogPath = os.Getenv("LOG_PATH")
	cfg.AppPort = os.Getenv("APP_PORT")
	return &cfg
}

func (cfg *Config) getEnvInt(key string, def int) int {
	env, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		log.Printf("Invalid %s environment variable, %s set to %d\n", key, key, def)
		env = def
	}
	return env
}

package main

import (
	"log"
	"os"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		pwd, _ := os.Getwd()
		log.Println("No .env file found in ", pwd)
	}
}

type Config struct {
	URI                 string `env:"URI" envDefault:"http://localhost:8080/"`
	ListenPort          int    `env:"LISTEN_PORT" envDefault:"8080"`
	KeyPart             string `env:"KEY_PART"`
	PersistentStorage   bool   `env:"PERSISTENT_STORAGE" envDefault:"false"`
	StoragePath         string `env:"STORAGE_PATH" envDefault:"data"`
	IDLength            int    `env:"ID_LENGTH" envDefault:"8"`
	KeyLength           int    `env:"KEY_LENGTH" envDefault:"8"`
	RunClearingInterval int    `env:"RUN_CLEARING_INTERVAL" envDefault:"1800"`
	SecretsExpire       int    `env:"SECRETS_EXPIRE" envDefault:"86400"`
}

func ReadConfig() *Config {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatal("Error parsing ENV: ", err)
	}
	return &cfg
}

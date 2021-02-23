package main

import (
	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
	"log"
	"os"
)

func init() {
	if err := godotenv.Load(); err != nil {
		pwd, _ := os.Getwd()
		log.Println("No .env file found in ", pwd)
	}
}

type Config struct {
	ListenPort        int    `env:"LISTEN_PORT" envDefault:"8834"`
	KeyPart           string `env:"KEY_PART"`
	PersistentStorage bool   `env:"PERSISTENT_STORAGE" envDefault:"false"`
	StoragePath       string `env:"STORAGE_PATH"`
	IdLength int `env:"ID_LENGTH" envDefault:"8"`
	KeyLength int `env:"KEY_LENGTH" envDefault:"8"`
}

func ReadConfig() *Config {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatal("Error parsing ENV", err)
	}
	return &cfg
}

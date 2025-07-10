package config

import (
	"log"

	"github.com/joho/godotenv"
)

func LoadEnvOrFail() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
}

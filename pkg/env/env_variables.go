package env

import (
	"log"

	"github.com/joho/godotenv"
)

// Загрузка переменных окружения из файла .env
func LoadEnvVariables() {
	err := godotenv.Load()
	if err != nil {
		log.Print("no .env file found")
	}
}

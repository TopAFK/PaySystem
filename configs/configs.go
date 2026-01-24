package configs

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

const (
	REQUEST_RATE = 10 * time.Second
)

var (
	BANK_ACCOUNT  string
	BANK_WUID     string
	BANK_CATEGORY string
	BANK_HOST     string
	BANK_PATH     string

	SYSTEM_KEY  string
	SYSTEM_HOST string
	SYSTEM_PATH string
)

func Init() error {
	if err := godotenv.Load("../../configs/.env"); err != nil {
		return err
	}

	BANK_ACCOUNT = os.Getenv("BANK_ACCOUNT")
	BANK_WUID = os.Getenv("BANK_WUID")
	BANK_CATEGORY = os.Getenv("BANK_CATEGORY")
	BANK_HOST = os.Getenv("BANK_HOST")
	BANK_PATH = os.Getenv("BANK_PATH")

	SYSTEM_KEY = os.Getenv("SYSTEM_KEY")
	SYSTEM_HOST = os.Getenv("SYSTEM_HOST")
	SYSTEM_PATH = os.Getenv("SYSTEM_PATH")

	return nil
}

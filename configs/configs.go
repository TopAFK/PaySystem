package configs

import (
	"fmt"
	"os"
	"strings"
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

	TBANK_PHONE    string
	TBANK_PIN      string
	TBANK_PASSWORD string

	SMS_CODE_FILE string

	DB_DSN string
)

func Init() error {
	if envFile := strings.TrimSpace(os.Getenv("ENV_FILE")); envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			return err
		}
	} else {
		_ = loadEnvFileOptional("../../configs/.env")
		_ = loadEnvFileOptional(".env")
		_ = loadEnvFileOptional("/configs/.env")
	}

	BANK_ACCOUNT = os.Getenv("BANK_ACCOUNT")
	BANK_WUID = os.Getenv("BANK_WUID")
	BANK_CATEGORY = os.Getenv("BANK_CATEGORY")
	BANK_HOST = os.Getenv("BANK_HOST")
	BANK_PATH = os.Getenv("BANK_PATH")

	SYSTEM_KEY = os.Getenv("SYSTEM_KEY")
	SYSTEM_HOST = os.Getenv("SYSTEM_HOST")
	SYSTEM_PATH = os.Getenv("SYSTEM_PATH")

	TBANK_PHONE = os.Getenv("TBANK_PHONE")
	TBANK_PIN = os.Getenv("TBANK_PIN")
	TBANK_PASSWORD = os.Getenv("TBANK_PASSWORD")

	SMS_CODE_FILE = os.Getenv("SMS_CODE_FILE")

	DB_DSN = os.Getenv("DB_DSN")

	missing := []string{}
	if BANK_HOST == "" {
		missing = append(missing, "BANK_HOST")
	}
	if BANK_PATH == "" {
		missing = append(missing, "BANK_PATH")
	}
	if SYSTEM_KEY == "" {
		missing = append(missing, "SYSTEM_KEY")
	}
	if SYSTEM_HOST == "" {
		missing = append(missing, "SYSTEM_HOST")
	}
	if SYSTEM_PATH == "" {
		missing = append(missing, "SYSTEM_PATH")
	}
	if TBANK_PHONE == "" {
		missing = append(missing, "TBANK_PHONE")
	}
	if TBANK_PIN == "" {
		missing = append(missing, "TBANK_PIN")
	}
	if TBANK_PASSWORD == "" {
		missing = append(missing, "TBANK_PASSWORD")
	}
	if SMS_CODE_FILE == "" {
		missing = append(missing, "SMS_CODE_FILE")
	}
	if DB_DSN == "" {
		missing = append(missing, "DB_DSN")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}

	return nil
}

func loadEnvFileOptional(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return godotenv.Load(path)
}

func parseDurationEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	durationValue, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	return durationValue, nil
}

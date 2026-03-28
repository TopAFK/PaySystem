package sms

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Response struct {
	SMSCode string `json:"sms_code"`
}

// FetchCode — получает код из файла (если задан) или через HTTP.
func GetCode(codeFile string, pollInterval time.Duration, timeout time.Duration) (string, error) {
	return waitForCodeFile(codeFile, pollInterval, timeout)
}

func waitForCodeFile(codeFile string, pollInterval time.Duration, timeout time.Duration) (string, error) {
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}

	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return "", fmt.Errorf("timeout waiting for SMS code in %s", codeFile)
		}

		content, err := os.ReadFile(codeFile)
		if err != nil {
			if os.IsNotExist(err) {
				time.Sleep(pollInterval)
				continue
			}
			return "", err
		}

		code := strings.TrimSpace(string(content))
		if code != "" {
			_ = os.WriteFile(codeFile, []byte{}, 0o600)
			return code, nil
		}

		time.Sleep(pollInterval)
	}
}

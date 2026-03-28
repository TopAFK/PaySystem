package tbank

import (
    "time"

    "github.com/pquerna/otp/totp"
)

func GenerateTOTP(secret string) (string, error) {
    return totp.GenerateCode(secret, time.Now())
}
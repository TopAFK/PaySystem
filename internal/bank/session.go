package bank

import (
	"encoding/json"
	"errors"
	"os/exec"
	"paysystem/pkg/chrome"
	"time"

	"github.com/go-vgo/robotgo"
)

type Session struct {
	Value  string `json:"value"`
	Status string `json:"status"`
}

const (
	SessionStatusOK = "OK"
)

func GetSession() (*Session, error) {
	getSessionID := exec.Command("./../bin/getSessionID")

	output, err := getSessionID.Output()
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(output, &session); err != nil {
		return nil, err
	}

	if session.Status != SessionStatusOK {
		return nil, errors.New("session status is not OK")
	}

	return &session, nil
}

func UpdateSession() error {
	url := "https://www.tbank.ru/login"

	if err := chrome.Open(url); err != nil {
		return err
	}
	time.Sleep(time.Second * 15)

	robotgo.TypeStr("1010")

	time.Sleep(time.Second * 15)

	return nil
}
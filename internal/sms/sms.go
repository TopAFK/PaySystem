package sms

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/valyala/fasthttp"
)

type Response struct {
	SMSCode string `json:"sms_code"`
}

// FetchCode — запрашивает у пользователя код из СМС.
func FetchCode(systemKey string, systemHost string, systemPath string) (string, error) {
	var uri fasthttp.URI
	uri.SetScheme("https")
	uri.SetHost(systemHost)
	uri.SetPath(systemPath)

	q := uri.QueryArgs()
	q.Add("do", "sms_code")
	q.Add("key", systemKey)

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(uri.String())
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/x-www-form-urlencoded")

	req.SetBody([]byte(q.String()))

	if err := fasthttp.DoTimeout(req, resp, time.Minute); err != nil {
		return "", err
	}

	if statusCode := resp.StatusCode(); statusCode != fasthttp.StatusOK {
		return "", fmt.Errorf("HTTP %d", statusCode)
	}

	var response Response
	if err := json.Unmarshal(resp.Body(), &response); err != nil {
		return "", err
	}

	return response.SMSCode, nil
}

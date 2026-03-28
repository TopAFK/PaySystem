package tbank

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"github.com/valyala/fasthttp"
)

type Payment struct {
	Text   string `json:"text"`
	Status string `json:"status"`
}

type Response struct {
	ResultCode string      `json:"resultCode"`
	Payload    []Operation `json:"payload"`
}

type Operation struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Amount    Amount    `json:"amount"`
	CreatedAt CreatedAt `json:"operationTime"`
}

type CreatedAt struct {
	Milliseconds int64 `json:"milliseconds"`
}

type Amount struct {
	Sum decimal.Decimal `json:"value"`
}

func FetchData(bankSessionID string, bankHost string, bankPath string, requestRate time.Duration, client *fasthttp.Client) (Response, error) {
	now := time.Now().UTC()
	end := now.UnixMilli()
	start := now.Add(-24 * time.Hour).UnixMilli()

	var apiResponse Response
	var uri fasthttp.URI
	uri.SetScheme("https")
	uri.SetHost(bankHost)
	uri.SetPath(bankPath)

	q := uri.QueryArgs()
	q.Add("sessionid", bankSessionID)
	q.Add("end", strconv.FormatInt(end, 10))
	q.Add("start", strconv.FormatInt(start, 10))

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(uri.String())
	req.Header.SetMethod(fasthttp.MethodGet)

	if err := client.DoTimeout(req, resp, requestRate); err != nil {
		return apiResponse, err
	}

	statusCode := resp.StatusCode()
	if statusCode != fasthttp.StatusOK {
		return apiResponse, fmt.Errorf("HTTP %d: %s", statusCode, resp.Body())
	}

	if err := json.Unmarshal(resp.Body(), &apiResponse); err != nil {
		return apiResponse, err
	}
	return apiResponse, nil
}

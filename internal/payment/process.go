package payment

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"github.com/valyala/fasthttp"
)

type Status string

const (
	StatusSucceeded Status = "succeeded"
	StatusDuplicate Status = "duplicate"
	StatusError     Status = "error"
)

type Response struct {
	Text   string `json:"text"`
	Status Status `json:"status"`
}

func Process(systemKey string, systemHost string, systemPath string, paidAt int64, sum decimal.Decimal) (*Response, error) {
	var uri fasthttp.URI
	uri.SetScheme("https")
	uri.SetHost(systemHost)
	uri.SetPath(systemPath)

	q := uri.QueryArgs()
	q.Add("do", "pay_payment_v2")
	q.Add("key", systemKey)
	q.Add("paid_at", strconv.FormatInt(paidAt, 10))
	q.Add("sum", sum.StringFixed(2))

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(uri.String())
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/x-www-form-urlencoded")

	req.SetBody([]byte(q.String()))

	if err := fasthttp.DoTimeout(req, resp, time.Minute); err != nil {
		return nil, err
	}

	if statusCode := resp.StatusCode(); statusCode != fasthttp.StatusOK {
		return nil, fmt.Errorf("HTTP %d", statusCode)
	}

	var response Response
	if err := json.Unmarshal(resp.Body(), &response); err != nil {
		return nil, err
	}

	return &response, nil
}

package httpx

import (
	"time"

	"github.com/valyala/fasthttp"
)

type Client = fasthttp.Client

func New(timeout time.Duration) *Client {
	return &fasthttp.Client{
		ReadTimeout:                   timeout,
		WriteTimeout:                  timeout,
		MaxConnsPerHost:               128,
		MaxIdleConnDuration:           90 * time.Second,
		NoDefaultUserAgentHeader:      false,
		DisableHeaderNamesNormalizing: false,
	}
}

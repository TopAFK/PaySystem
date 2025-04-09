package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
)

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
	Sum float64 `json:"value"`
}

type Payment struct {
	Text   string `json:"text"`
	Status string `json:"status"`
}

const (
	parallelism = 4
	requestRate = 5 * time.Second

	ResultOK                     = "OK"
	ResultInsufficientPrivileges = "INSUFFICIENT_PRIVILEGES"

	StatusPaid  = "paid"
	StatusMade  = "made"
	StatusError = "error"

	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
)

var (
	// Конфигурация для bank
	bankAccount   string
	bankWuid      string
	bankCategory  string
	bankSessionid string
	bankHost      string
	bankPath      string

	// Конфигурация для системы
	systemKey  string
	systemHost string
	systemPath string

	client              = &fasthttp.Client{MaxConnsPerHost: parallelism}
	jobChan             = make(chan struct{}, parallelism)
	processedPaymentIDs []string
)

func contains(arr []string, value string) bool {
	for _, v := range arr {
		if v == value {
			return true
		}
	}
	return false
}

func ProcessPayment(paidAt int64, sum float64) {
	var uri fasthttp.URI
	uri.SetScheme("https")
	uri.SetHost(systemHost)
	uri.SetPath(systemPath)

	q := uri.QueryArgs()
	q.Add("do", "pay_payment_v2")
	q.Add("key", systemKey)
	q.Add("paid_at", strconv.FormatInt(paidAt, 10))
	q.Add("sum", strconv.FormatFloat(sum, 'f', 2, 64))

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(uri.String())
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/x-www-form-urlencoded")

	req.SetBody([]byte(q.String()))

	if err := fasthttp.Do(req, resp); err != nil {
		log.Println("Ошибка запроса:", err)
		return
	}

	statusCode := resp.StatusCode()
	if statusCode != fasthttp.StatusOK {
		log.Printf("⚠️ HTTP %d: %s\n", statusCode, resp.Body())
		return
	}

	var payment Payment
	if err := json.Unmarshal(resp.Body(), &payment); err != nil {
		log.Println("Ошибка декодирования JSON:", err)
		return
	}

	file, err := os.OpenFile("payments.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	logger := log.New(file, "", log.LstdFlags|log.Lshortfile)

	details := fmt.Sprintf("%s SUM: %-.2f DATE: %-d", payment.Text, sum, paidAt)
	switch payment.Status {
	case StatusMade:
		log.Println(green + details + reset)
		logger.Println("[INFO]", details)
	case StatusPaid:
		log.Println(yellow + details + reset)
		logger.Println("[WARN]", details)
	case StatusError:
		log.Println(red + details + reset)
		logger.Println("[ERROR]", details)
	default:
		log.Println(red + "Unknown error" + reset)
		logger.Println("[ERROR] Unknown error")
	}

}

func fetchData() {
	defer func() { <-jobChan }()

	now := time.Now().UTC()
	end := now.UnixMilli()
	start := now.Add(-24 * time.Hour).UnixMilli()

	var uri fasthttp.URI
	uri.SetScheme("https")
	uri.SetHost(bankHost)
	uri.SetPath(bankPath)

	q := uri.QueryArgs()
	q.Add("start", strconv.FormatInt(start, 10))
	q.Add("end", strconv.FormatInt(end, 10))
	q.Add("account", bankAccount)
	q.Add("spendingCategory", bankCategory)
	q.Add("sessionid", bankSessionid)
	q.Add("wuid", bankWuid)

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(uri.String())
	req.Header.SetMethod(fasthttp.MethodGet)
	//req.Header.Set("Authorization", "Bearer "+key) // если не нужно — закомментируй

	if err := client.DoTimeout(req, resp, requestRate); err != nil {
		log.Println("Ошибка запроса:", err)
		return
	}

	statusCode := resp.StatusCode()
	if statusCode != fasthttp.StatusOK {
		log.Printf("⚠️ HTTP %d: %s\n", statusCode, resp.Body())
		return
	}

	var apiResponse Response
	if err := json.Unmarshal(resp.Body(), &apiResponse); err != nil {
		log.Println("Ошибка декодирования JSON:", err)
		return
	}

	if apiResponse.ResultCode != ResultOK {
		switch apiResponse.ResultCode {
		case ResultInsufficientPrivileges:
			log.Println("⚠️ Сессия устарела")
			return
		default:
			log.Printf("⚠️ Не известная ошибка, код %s", apiResponse.ResultCode)
			return
		}
	}

	for _, op := range apiResponse.Payload {
		if op.Type != "Credit" {
			continue
		}

		if contains(processedPaymentIDs, op.ID) {
			continue
		}

		processedPaymentIDs = append(processedPaymentIDs, op.ID)
		paidAt := op.CreatedAt.Milliseconds / 1000
		sum := op.Amount.Sum
		go ProcessPayment(paidAt, sum)
	}
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Ошибка загрузки .env файла:", err)
	}

	bankAccount = os.Getenv("BANK_ACCOUNT")
	bankWuid = os.Getenv("BANK_WUID")
	bankCategory = os.Getenv("BANK_CATEGORY")
	bankSessionid = os.Getenv("BANK_SESSIONID")
	bankHost = os.Getenv("BANK_HOST")
	bankPath = os.Getenv("BANK_PATH")

	systemKey = os.Getenv("SYSTEM_KEY")
	systemHost = os.Getenv("SYSTEM_HOST")
	systemPath = os.Getenv("SYSTEM_PATH")
}

func main() {
	ticker := time.NewTicker(requestRate)
	defer ticker.Stop()

	log.Println("🚀 Запуск клиента...")

	for {
		select {
		case jobChan <- struct{}{}:
			go fetchData()
		default:
			log.Println("⏳ Пропуск: все воркеры заняты")
		}

		<-ticker.C
	}

}

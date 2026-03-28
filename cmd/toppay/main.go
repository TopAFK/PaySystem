package main

import (
	"log"
	"time"

	"paysystem/configs"
	"paysystem/internal/payment"
	"paysystem/pkg/db"
	"paysystem/pkg/httpx"
	"paysystem/pkg/tbank"
)

var (
	processedPaymentIDs = make(map[string]struct{})
)

const (
	Online = iota
	Offline
)

const (
	ResultOK                     = "OK"
	ResultInsufficientPrivileges = "AUTHENTICATION_FAILED"

	OpTypeCredit = "Credit"
)

func main() {

	log.Println("1/4 ▫️ Starting client")

	
	log.Println("2/4 ▫️ Initializing configs")
	if err := configs.Init(); err != nil {
		log.Fatal(err)
	}

	log.Println("3/4 ▫️ Initializing database")

	if err := db.Init(configs.DB_DSN); err != nil {
		log.Fatal(err)
	}

	log.Println("4/4 ▫️ Fetching session ID")

	const maxRetries = 5
	var bankSession string
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		bankSession, err = tbank.GetSession()
		if err == nil {
			break
		}

		log.Printf("❌ Error fetching session ID, attempt %d/%d: %v", attempt, maxRetries, err)

		if attempt < maxRetries {
			time.Sleep(2 * time.Second) // Можно добавить задержку перед следующей попыткой
		}
	}

	if err != nil {
		log.Fatal("❌ Failed to get session ID after ", maxRetries, " attempts: ", err)
	}
	
	client := httpx.New(configs.REQUEST_RATE)

	for {
		for {
			log.Println("Fetching data")

			response, err := tbank.FetchData(bankSession, configs.BANK_HOST, configs.BANK_PATH, configs.REQUEST_RATE, client)
			if err != nil {
				log.Println("❌ Error fetching data:", err)
				time.Sleep(time.Second)
				continue
			}

			if response.ResultCode != ResultOK {
				switch response.ResultCode {
				case ResultInsufficientPrivileges:
					
					log.Println("Updating session")
					bankSession, err = tbank.GetSession()
					if err != nil {
						log.Println("❌ Error updating session:", err)
						continue
					}

					log.Println("✅ Session is updated")
				default:
					log.Println("❌ Unknown error, code:", response.ResultCode)
				}
				time.Sleep(time.Second)
				continue
			}

			for _, op := range response.Payload {
				if op.Type != OpTypeCredit {
					continue
				}

				if _, exists := processedPaymentIDs[op.ID]; exists {
					continue
				}

				paidAt := op.CreatedAt.Milliseconds / 1000
				sum := op.Amount.Sum

				log.Printf("Processing payment")

				status, err := payment.Process(db.TD, paidAt, sum)

				if err != nil {
					log.Println("❌ Error processing payment:", err)
					continue
				}

				processedPaymentIDs[op.ID] = struct{}{}
				switch status {
				case payment.StatusSucceeded:
					log.Printf("✅ Payment succeeded, sum: %s", sum)
				case payment.StatusDuplicate:
					log.Printf("⚠️ Payment duplicate, sum: %s", sum)
				case payment.StatusError:
					log.Printf("❌ Payment error, sum: %s", sum)
				}
			}

			break
		}
		time.Sleep(configs.REQUEST_RATE)
	}
}
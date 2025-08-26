package main

import (
	"fmt"
	"log"
	"time"

	"paysystem/configs"
	"paysystem/internal/bank"
	"paysystem/internal/payment"
	"paysystem/pkg/logger"

	"github.com/valyala/fasthttp"
)

var (
	client              = &fasthttp.Client{}
	processedPaymentIDs = make(map[string]struct{})
)

const (
	ResultOK                     = "OK"
	ResultInsufficientPrivileges = "INSUFFICIENT_PRIVILEGES"

	OpTypeCredit = "Credit"
)

func main() {
	log.Println("1/5 🆗 Starting client...")
	if err := logger.Init(); err != nil {
		log.Fatal(err)
	}
	log.Println("2/5 🆗 Logger initialized")

	if err := config.Init(); err != nil {
		log.Fatal(err)
	}
	log.Println("3/5 🆗 Config initialized")

	log.Println("4/5 🆗 Fetching session ID...")
	bankSession, err := bank.GetSession()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("5/5 🆗 Session ID obtained")

	for {
		for {
			log.Println("🌀 Fetching data...")

			response, err := bank.FetchData(bankSession.Value, config.BANK_HOST, config.BANK_PATH, config.REQUEST_RATE, client)
			
			fmt.Print("\033[1A\033[2K")

			if err != nil {
				log.Println("❌ Error fetching data:", err)
				time.Sleep(time.Second)
				continue
			}


			//log.Println("Response:", bankSession.Value)
			//log.Println("Response:", response)

			if response.ResultCode != ResultOK {
				switch response.ResultCode {
				case ResultInsufficientPrivileges:
					log.Println("⚠️ Session is outdated")
					log.Println("🌀 Updating session ID...")
					if err := bank.UpdateSession(); err != nil {
						log.Println("❌ Error updating session ID:", err)
					}
					bankSession, err = bank.GetSession()
					if err != nil {
						log.Println(err)
					}
					fmt.Print("\033[1A\033[2K")

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

				log.Printf("🌀 Processing payment...")

				response, err := payment.Process(config.SYSTEM_KEY, config.SYSTEM_HOST, config.SYSTEM_PATH, paidAt, sum)

				fmt.Print("\033[1A\033[2K")
				if err != nil {
					log.Println("❌ Error processing payment:", err)
					continue
				}

				processedPaymentIDs[op.ID] = struct{}{}
				switch response.Status {
				case payment.StatusSucceeded:
					log.Printf("✅ Payment succeeded, sum: %s", sum)
				case payment.StatusDuplicate:
					log.Printf("⚠️ Payment duplicate, sum: %s", sum)
				case payment.StatusError:
					log.Printf("❌ Payment error, sum: %s, message: %s", sum, response.Text)
				}
			}

			break
		}
		time.Sleep(config.REQUEST_RATE)
	}
}



                               






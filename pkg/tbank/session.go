package tbank

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
	"toppay/configs"

	"github.com/chromedp/chromedp"
)

const (
	// Tbank login URL
	TbankLoginURL = "https://www.tbank.ru/login"
	// Tbank operations/history page
	TbankOperationsURL = "https://www.tbank.ru/mybank/operations"
	// Timeout для всей сессии браузера
	BrowserSessionTimeout = 120 * time.Second
	// Timeout между TOTP и следующим шагом
	PageReactionTimeout = 2 * time.Second
	// Timeout для финального редиректа после входа
	FinalRedirectTimeout = 3 * time.Second
)

func GetSession() (string, error) {
	// Создаём контекст с таймаутом для всей операции
	ctx, cancel := context.WithTimeout(context.Background(), BrowserSessionTimeout)
	defer cancel()

	// Создаём chromedp контекст с headless флагом и custom user-agent
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Headless,
		chromedp.NoSandbox,
		chromedp.DisableGPU,
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	log.Println("1/5 Launching browser")

	// Отключаем логирование ошибок от chromedp (DevTools Protocol parsing errors)
	chromeCtx, cancel := chromedp.NewContext(allocCtx,
		chromedp.WithLogf(func(string, ...interface{}) {}),   // отключить логи
		chromedp.WithErrorf(func(string, ...interface{}) {}), // отключить ошибки
	)
	defer cancel()

	log.Println("2/5 Navigating to login page")

	// Получаем конфиги
	phoneMasked := configs.TBANK_PHONE
	pinCode := configs.TBANK_PIN
	password := configs.TBANK_PASSWORD

	// Логика навигации и входа
	var psidCookie string
	err := chromedp.Run(chromeCtx,
		chromedp.Navigate(TbankLoginURL),
		chromedp.WaitVisible("[automation-id=phone-input], #pinCode0", chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Проверяем, видимо ли поле телефона
			var isPhoneVisible bool
			if err := chromedp.Evaluate(`document.querySelector('[automation-id=phone-input]') !== null`, &isPhoneVisible).Do(ctx); err != nil {
				return err
			}

			if isPhoneVisible {
				log.Println("1/9 Entering phone")
				if err := chromedp.SendKeys("[automation-id=phone-input]", phoneMasked, chromedp.ByQuery).Do(ctx); err != nil {
					return fmt.Errorf("enter phone: %w", err)
				}

				log.Println("2/9 Submitting phone")
				if err := chromedp.Click("[automation-id=button-submit]", chromedp.ByQuery).Do(ctx); err != nil {
					return fmt.Errorf("click submit: %w", err)
				}

				// Ждём появления поля для TOTP
				log.Println("3/9 Waiting for TOTP")
				if err := chromedp.WaitVisible("#pinCode0", chromedp.ByQuery).Do(ctx); err != nil {
					return fmt.Errorf("wait pin code 0: %w", err)
				}

				totp, err := GenerateTOTP(configs.TBANK_TOTP_SECRET)
				if err != nil {
					return fmt.Errorf("GenerateTOTP: %w", err)
				}

				log.Println("4/9 Setting TOTP")
				if err := setPinCodeChromedp(ctx, totp); err != nil {
					return fmt.Errorf("setPinCode (totp): %w", err)
				}

				// Даём время на обработку
				log.Println("5/9 Waiting for page reaction")
				if err := chromedp.Sleep(PageReactionTimeout).Do(ctx); err != nil {
					return fmt.Errorf("sleep after TOTP: %w", err)
				}

				// Проверяем, видимо ли поле пароля
				var isPasswordVisible bool
				if err := chromedp.Evaluate(`document.querySelector('[automation-id=password-input]') !== null && window.getComputedStyle(document.querySelector('[automation-id=password-input]')).display !== 'none'`, &isPasswordVisible).Do(ctx); err == nil && isPasswordVisible {
					log.Println("6/9 Setting password")
					if err := chromedp.SendKeys("[automation-id=password-input]", password, chromedp.ByQuery).Do(ctx); err != nil {
						return fmt.Errorf("enter password: %w", err)
					}

					log.Println("7/9 Submitting password")
					if err := chromedp.Click("[automation-id=button-submit]", chromedp.ByQuery).Do(ctx); err != nil {
						return fmt.Errorf("click submit (after password): %w", err)
					}

					// Жёдем времени для обновления страницы
					log.Println("Waiting for page update after password submit...")
					if err := chromedp.Sleep(PageReactionTimeout).Do(ctx); err != nil {
						return fmt.Errorf("sleep after password submit: %w", err)
					}
					log.Println("Page update completed, proceeding to PIN code entry")
				}

				log.Println("8/9 Starting PIN code entry")
				if err := setPinCodeChromedp(ctx, pinCode); err != nil {
					return fmt.Errorf("setPinCode (pin): %w", err)
				}
				log.Println("8/9 PIN code entered successfully")

				log.Println("9/9 Clicking final submit button")
				if err := chromedp.Click("[automation-id=button-submit]", chromedp.ByQuery).Do(ctx); err != nil {
					return fmt.Errorf("click submit (final): %w", err)
				}
				log.Println("9/9 Final submit button clicked")

				// Жёдем загрузку страницы после редиректа
				log.Println("Waiting for page load after final submit...")
				if err := chromedp.Sleep(FinalRedirectTimeout).Do(ctx); err != nil {
					log.Println("ERROR during sleep: ", err)
					return fmt.Errorf("sleep after final submit: %w", err)
				}
				log.Println("Page load wait completed")
			} else {
				// Телефон не спрашивали — возможно, сразу пин-код
				log.Println("1/1 Setting PIN code (no phone step)")
				if err := setPinCodeChromedp(ctx, pinCode); err != nil {
					return fmt.Errorf("setPinCode (no phone step): %w", err)
				}
			}
			log.Println("Login flow completed successfully")
			return nil
		}),
	)
	if err != nil {
		return "", fmt.Errorf("login flow: %v", err)
	}

	log.Println("3/5 Navigating to operations page")
	err = chromedp.Run(chromeCtx,
		chromedp.Navigate(TbankOperationsURL),
		chromedp.Sleep(PageReactionTimeout), // Даём время на редиректы
	)
	if err != nil {
		return "", fmt.Errorf("navigate to operations: %v", err)
	}

	log.Println("4/5 Fetching cookies")
	err = chromedp.Run(chromeCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Получаем все cookies через JavaScript
			var cookies []map[string]interface{}
			if err := chromedp.Evaluate(`
				(() => {
					return document.cookie.split(';').map(c => {
						const [name, ...rest] = c.trim().split('=');
						return {name: name, value: rest.join('=')};
					});
				})()
			`, &cookies).Do(ctx); err != nil {
				return fmt.Errorf("get cookies: %w", err)
			}

			for _, c := range cookies {
				if nameVal, ok := c["name"].(string); ok && nameVal == "psid" {
					if valueVal, ok := c["value"].(string); ok {
						log.Println("5/5 ✅ Cookie found")
						log.Println(valueVal)
						psidCookie = valueVal
						return nil
					}
				}
			}
			return fmt.Errorf("psid cookie not found")
		}),
	)
	if err != nil {
		return "", err
	}

	return psidCookie, nil
}

// setPinCodeChromedp вводит пин-код по полям с id вида pinCode0..pinCodeN используя chromedp
func setPinCodeChromedp(ctx context.Context, pin string) error {
	digits := strings.Split(pin, "")
	for i, digit := range digits {
		sel := fmt.Sprintf("#pinCode%d", i)

		// Ждём видимости элемента
		if err := chromedp.WaitVisible(sel, chromedp.ByQuery).Do(ctx); err != nil {
			return fmt.Errorf("wait for selector %s: %w", sel, err)
		}

		// Очищаем поле и вводим символ
		if err := chromedp.SendKeys(sel, digit, chromedp.ByQuery).Do(ctx); err != nil {
			return fmt.Errorf("fill digit at %s: %w", sel, err)
		}
	}
	return nil
}

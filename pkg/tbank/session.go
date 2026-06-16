package tbank

import (
	"context"
	"fmt"
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

	// Отключаем логирование ошибок от chromedp (DevTools Protocol parsing errors)
	chromeCtx, cancel := chromedp.NewContext(allocCtx,
		chromedp.WithLogf(func(string, ...interface{}) {}),   // отключить логи
		chromedp.WithErrorf(func(string, ...interface{}) {}), // отключить ошибки
	)
	defer cancel()

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
				if err := chromedp.SendKeys("[automation-id=phone-input]", phoneMasked, chromedp.ByQuery).Do(ctx); err != nil {
					return fmt.Errorf("enter phone: %w", err)
				}

				if err := chromedp.Click("[automation-id=button-submit]", chromedp.ByQuery).Do(ctx); err != nil {
					return fmt.Errorf("click submit: %w", err)
				}

				// Ждём появления поля для TOTP
				if err := chromedp.WaitVisible("#pinCode0", chromedp.ByQuery).Do(ctx); err != nil {
					return fmt.Errorf("wait pin code 0: %w", err)
				}

				totp, err := GenerateTOTP(configs.TBANK_TOTP_SECRET)
				if err != nil {
					return fmt.Errorf("GenerateTOTP: %w", err)
				}

				if err := setPinCodeChromedp(ctx, totp); err != nil {
					return fmt.Errorf("setPinCode (totp): %w", err)
				}

				// Даём время на обработку
				if err := chromedp.Sleep(PageReactionTimeout).Do(ctx); err != nil {
					return fmt.Errorf("sleep after TOTP: %w", err)
				}

				// Проверяем, видимо ли поле пароля
				var isPasswordVisible bool
				if err := chromedp.Evaluate(`document.querySelector('[automation-id=password-input]') !== null && window.getComputedStyle(document.querySelector('[automation-id=password-input]')).display !== 'none'`, &isPasswordVisible).Do(ctx); err == nil && isPasswordVisible {
					if err := chromedp.SendKeys("[automation-id=password-input]", password, chromedp.ByQuery).Do(ctx); err != nil {
						return fmt.Errorf("enter password: %w", err)
					}

					if err := chromedp.Click("[automation-id=button-submit]", chromedp.ByQuery).Do(ctx); err != nil {
						return fmt.Errorf("click submit (after password): %w", err)
					}

					// Жёдем времени для обновления страницы
					if err := chromedp.Sleep(PageReactionTimeout).Do(ctx); err != nil {
						return fmt.Errorf("sleep after password submit: %w", err)
					}
				}

				if err := setPinCodeChromedp(ctx, pinCode); err != nil {
					return fmt.Errorf("setPinCode (pin): %w", err)
				}

				if err := chromedp.Click("[automation-id=button-submit]", chromedp.ByQuery).Do(ctx); err != nil {
					return fmt.Errorf("click submit (final): %w", err)
				}

				// Жёдем загрузку страницы после редиректа
				if err := chromedp.Sleep(FinalRedirectTimeout).Do(ctx); err != nil {
					return fmt.Errorf("sleep after final submit: %w", err)
				}
			} else {
				// Телефон не спрашивали — возможно, сразу пин-код
				if err := setPinCodeChromedp(ctx, pinCode); err != nil {
					return fmt.Errorf("setPinCode (no phone step): %w", err)
				}
			}
			return nil
		}),
	)
	if err != nil {
		return "", fmt.Errorf("login flow: %v", err)
	}

	err = chromedp.Run(chromeCtx,
		chromedp.Navigate(TbankOperationsURL),
		chromedp.Sleep(PageReactionTimeout), // Даём время на редиректы
	)
	if err != nil {
		return "", fmt.Errorf("navigate to operations: %v", err)
	}

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

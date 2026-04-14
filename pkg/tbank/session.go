package tbank

import (
	"fmt"
	"log"
	"strings"
	"toppay/configs"

	"github.com/playwright-community/playwright-go"
)

func GetSession() (string, error) {
	if err := playwright.Install(); err != nil {
		return "", fmt.Errorf("playwright.Install: %v", err)
	}

	log.Println("1/5 Launching Playwright")
	pw, err := playwright.Run()
	if err != nil {
		return "", fmt.Errorf("playwright.Run: %v", err)
	}
	defer func() { _ = pw.Stop() }()

	log.Println("2/5 Launching browser")
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		return "", fmt.Errorf("Launch: %v", err)
	}
	defer func() { _ = browser.Close() }()

	log.Println("3/5 Creating context")
	ctx, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"),
	})
	if err != nil {
		return "", fmt.Errorf("NewContext: %v", err)
	}
	defer func() { _ = ctx.Close() }()


	if err := ctx.Route("**", func(r playwright.Route) {
		switch r.Request().ResourceType() {
		case "image", "media", "font", "stylesheet":
			_ = r.Abort()
		default:
			_ = r.Continue()
		}
	}); err != nil {
		return "", fmt.Errorf("Route: %v", err)
	}

	log.Println("4/5 Creating page")

	page, err := ctx.NewPage()
	if err != nil {
		return "", fmt.Errorf("NewPage: %v", err)
	}

	// Таймауты
	page.SetDefaultTimeout(60_000)
	page.SetDefaultNavigationTimeout(100_000)

	log.Println("5/5 Navigating")

	waitState := playwright.WaitUntilState("networkidle")

	_, err = page.Goto("https://www.tbank.ru/login", playwright.PageGotoOptions{
		WaitUntil: &waitState,
	})
	if err != nil {
		return "", fmt.Errorf("Goto: %v", err)
	}

	waitPage(page)

	phoneMasked := configs.TBANK_PHONE
	pinCode := configs.TBANK_PIN
	password := configs.TBANK_PASSWORD

	log.Println("9/9 Handling login steps")
	// На странице либо есть ввод телефона, либо сразу пин-код.
	phoneInput := page.Locator("[automation-id=phone-input]")
	pinCodeInput0 := page.Locator("#pinCode0")
	submitBtn := page.Locator("[automation-id=button-submit]")
	passwordInput := page.Locator("[automation-id=password-input]")

	if visible, _ := phoneInput.IsVisible(); visible {

		// Вводим телефон
		log.Println("1/9 Entering phone")

		if err := phoneInput.Fill(phoneMasked); err != nil {
			return "", fmt.Errorf("Fill: %v", err)
		}

		// Подтверждаем
		log.Println("2/9 Submitting phone")

		if err := submitBtn.Click(); err != nil {
			return "", fmt.Errorf("Click submit: %v", err)
		}

		// Подстраховка — ждём networkidle, но полагаемся на селекторы:
		waitPage(page)

		// Ждём появления поля для TOTP
		log.Println("3/9 Waiting for TOTP")
		totp, err := GenerateTOTP(configs.TBANK_TOTP_SECRET)
		if err != nil {
			return "", fmt.Errorf("GenerateTOTP: %v", err)
		}

		log.Println("4/9 Setting TOTP")
		if err := setPinCode(page, totp); err != nil {
			return "", fmt.Errorf("setPinCode: %v", err)
		}

		// 3 После ввода TOTP может открыться экран с пин-кодом
		waitPage(page)

		err = passwordInput.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(3000),
			State:   playwright.WaitForSelectorStateVisible,
		})
		if err == nil {
			log.Println("6/9 Setting password")
			if err := passwordInput.Fill(password); err != nil {
				return "", fmt.Errorf("Fill password: %v", err)
			}

			// Подтверждаем
			log.Println("7/9 Submitting")
			if err := submitBtn.Click(); err != nil {
				return "", fmt.Errorf("Click submit (after password): %v", err)
			}
			waitPage(page)
		}

		log.Println("8/9 Setting PIN code")
		if err := setPinCode(page, pinCode); err != nil {
			return "", fmt.Errorf("setPinCode: %v", err)
		}

		// 4 Подтверждаем
		log.Println("9/9 Submitting")
		if err := submitBtn.Click(); err != nil {
			return "", fmt.Errorf("submitBtn.Click: %v", err)
		}

	} else if visible, _ := pinCodeInput0.IsVisible(); visible {
		// Телефон не спрашивали — возможно, сразу пин-код
		log.Println("1/1 Setting PIN code (no phone step)")
		if err := setPinCode(page, pinCode); err != nil {
			return "", fmt.Errorf("setPinCode (no phone step): %v", err)
		}
	}

	waitState = playwright.WaitUntilState("networkidle") // допустимое значение WaitUntil
	_, err = page.Goto("https://www.tbank.ru/mybank/operations", playwright.PageGotoOptions{
		WaitUntil: &waitState,
	})

	// Небольшая пауза — дождаться завершения редиректов/сетевых запросов
	log.Println("1/3 Waiting for redirects")
	waitPage(page)

	log.Println("2/3 Fetching cookies")

	cookies, err := ctx.Cookies()
	if err != nil {
		return "", fmt.Errorf("cookies: %v", err)
	}

	log.Println("3/3 Cookies searched")

	for _, c := range cookies {
		if c.Name == "psid" {
			log.Println("3/3 ✅ Cookies found")
			log.Println(c.Value)
			return c.Value, nil
		}
	}
	return "", fmt.Errorf("psid cookie not found")
}

// setPinCode вводит пин-код по полям с id вида pinCode0..pinCodeN
func setPinCode(page playwright.Page, pin string) error {
	// Некоторые страницы используют отдельные инпуты по символам:
	for i, ch := range strings.Split(pin, "") {
		sel := fmt.Sprintf("#pinCode%d", i)
		loc := page.Locator(sel)
		if err := loc.WaitFor(); err != nil {
			return fmt.Errorf("wait pin input %s: %w", sel, err)
		}
		if err := loc.Fill(ch); err != nil {
			return fmt.Errorf("fill pin %s: %w", sel, err)
		}
	}
	return nil
}

func waitPage(page playwright.Page) {
	_ = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})
}

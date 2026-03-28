package tbank

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"toppay/configs"

	"github.com/playwright-community/playwright-go"
)

var installOnce sync.Once

func GetSession() (string, error) {
	log.Println("1/9 Installing Playwright")
	/* var installErr error
	installOnce.Do(func() {
		installErr = playwright.Install()
	})
	if installErr != nil {
		return "", fmt.Errorf("playwright.Install: %v", installErr)
	} */
	if err := playwright.Install(); err != nil {
		return "", fmt.Errorf("playwright.Install: %v", err)
	}

	log.Println("2/7 Launching Playwright")
	pw, err := playwright.Run()
	if err != nil {
		return "", fmt.Errorf("playwright.Run: %v", err)
	}
	defer func() { _ = pw.Stop() }()

	// 1 задаёшь относительную папку профиля
	relDir := "./pw-profile"

	log.Println("3/9 Setting up profile")

	// 2 превращаешь в абсолютную
	userDataDir, err := filepath.Abs(relDir)
	if err != nil {
		return "", fmt.Errorf("filepath.Abs: %v", err)
	}

	log.Println("4/9 Ensuring profile exists")
	// 3 гарантируешь, что папка существует
	if err := os.MkdirAll(userDataDir, 0o755); err != nil {
		return "", fmt.Errorf("MkdirAll: %v", err)
	}

	log.Println("5/9 Launching persistent context")
	// 4 запускаешь persistent context
	ctx, err := pw.Chromium.LaunchPersistentContext(
		userDataDir,
		playwright.BrowserTypeLaunchPersistentContextOptions{
			Headless:  playwright.Bool(true),
			SlowMo:    playwright.Float(250),
			UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"),
		},
	)
	if err != nil {
		return "", fmt.Errorf("LaunchPersistentContext: %v", err)
	}

	defer func() { _ = ctx.Close() }()

	origins := []string{"https://www.tbank.ru", "https://id.tbank.ru"}

	for _, origin := range origins {
		err := ctx.GrantPermissions(
			[]string{"geolocation", "camera", "microphone", "notifications"},

			playwright.BrowserContextGrantPermissionsOptions{
				Origin: &origin,
			},
		)
		if err != nil {
			return "", fmt.Errorf("GrantPermissions for %s: %v", origin, err)
		}
	}

	var accuracy float64 = 50
	err = ctx.SetGeolocation(&playwright.Geolocation{
		Latitude:  55.7558,
		Longitude: 37.6173,
		Accuracy:  &accuracy,
	})

	if err != nil {
		return "", fmt.Errorf("SetGeolocation: %v", err)
	}

	log.Println("6/9 Route seting up")
	// Блокируем тяжёлые ресурсы (image/media/font/stylesheet)
	/* if err := ctx.Route("**", func(r playwright.Route) {
		switch r.Request().ResourceType() {
		case "image", "media", "font", "stylesheet":
			_ = r.Abort()
		default:
			_ = r.Continue()
		}
	}); err != nil {
		return "", fmt.Errorf("Route: %v", err)
	} */

	log.Println("7/9 Creating page")

	page, err := ctx.NewPage()
	if err != nil {
		return "", fmt.Errorf("NewPage: %v", err)
	}

	// Установим разумные таймауты для ожиданий
	page.SetDefaultTimeout(60_000)            // ожидания локаторов и т.п.
	page.SetDefaultNavigationTimeout(100_000) // навигации

	log.Println("8/9 Navigating to login page")
	// Навигация на страницу логина
	waitState := playwright.WaitUntilState("networkidle") // допустимое значение WaitUntil
	_, err = page.Goto("https://www.tbank.ru/login", playwright.PageGotoOptions{
		WaitUntil: &waitState, // см. предупреждение в доках — networkidle не для всех случаев
	})

	if err != nil {
		return "", fmt.Errorf("Goto: %v", err)
	}

	// Рекомендуемый способ «ожидания затухания сети» дополнительно:
	//_ = page.WaitForLoadState("networkidle") // строковое значение LoadState также поддерживается.
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
		WaitUntil: &waitState, // см. предупреждение в доках — networkidle не для всех случаев
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

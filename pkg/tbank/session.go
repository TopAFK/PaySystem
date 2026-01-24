package tbank

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"paysystem/internal/sms"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

func GetSession() (string, error) {
	log.Println(" 1/9 🌀 Installing Playwright...")
	if err := playwright.Install(); err != nil {
		return "", fmt.Errorf("playwright.Install: %v", err)
	}
	log.Println("\033[3A\033[K 1/9 🔸 Playwright installed")

	log.Println(" 2/7 🌀 Launching Playwright...")
	pw, err := playwright.Run()
	if err != nil {
		return "", fmt.Errorf("playwright.Run: %v", err)
	}
	log.Println("\033[1A\033[K 2/9 🔸 Playwright launched")
	defer func() { _ = pw.Stop() }()

	// 1) задаёшь относительную папку профиля
	relDir := "./pw-profile"

	log.Println(" 3/9 🌀 Setting up profile...")

	// 2) превращаешь в абсолютную
	userDataDir, err := filepath.Abs(relDir)
	if err != nil {
		return "", fmt.Errorf("filepath.Abs: %v", err)
	}
	log.Println("\033[1A\033[K 3/9 🔸 Profile set up at", userDataDir)

	log.Println(" 4/9 🌀 Ensuring profile exists...")
	// 3) гарантируешь, что папка существует
	if err := os.MkdirAll(userDataDir, 0o755); err != nil {
		return "", fmt.Errorf("MkdirAll: %v", err)
	}
	log.Println("\033[1A\033[K 4/9 🔸 Profile exists")

	log.Println(" 5/9 🌀 Launching persistent context...")
	// 4) запускаешь persistent context
	ctx, err := pw.Chromium.LaunchPersistentContext(
		userDataDir,
		playwright.BrowserTypeLaunchPersistentContextOptions{
			Headless:  playwright.Bool(true),
			SlowMo:    playwright.Float(250),
			UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"),
		},
	)
	log.Println("\033[1A\033[K 5/9 🔸 Persistent context launched")
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

	log.Println("\033[1A\033[K 6/9 🔸 Route set up")

	log.Println(" 7/9 🌀 Creating page...")


	page, err := ctx.NewPage()
	if err != nil {
		return "", fmt.Errorf("NewPage: %v", err)
	}
	log.Println("\033[1A\033[K 7/9 🔸 Page created")
	// Установим разумные таймауты для ожиданий
	page.SetDefaultTimeout(60_000)           // ожидания локаторов и т.п.
	page.SetDefaultNavigationTimeout(100_000) // навигации

	log.Println(" 8/9 🌀 Navigating to login page...")
	// Навигация на страницу логина
	waitState := playwright.WaitUntilState("networkidle") // допустимое значение WaitUntil
	_, err = page.Goto("https://www.tbank.ru/login", playwright.PageGotoOptions{
		WaitUntil: &waitState, // см. предупреждение в доках — networkidle не для всех случаев
	})
	log.Println("\033[1A\033[K 8/9 🔸 Login page navigated")
	if err != nil {
		return "", fmt.Errorf("Goto: %v", err)
	}

	// Рекомендуемый способ «ожидания затухания сети» дополнительно:
	//_ = page.WaitForLoadState("networkidle") // строковое значение LoadState также поддерживается.
	waitPage(page)

	// Данные — задай свои
	const (
		phoneMasked = "9963204601"
		pinCode     = "1010"
		password    = "QbZ$WK6k&-w71"
	)

	log.Println(" 9/9 🌀 Handling login steps...")
	// На странице либо есть ввод телефона, либо сразу пин-код.
	phoneInput := page.Locator("[automation-id=phone-input]")
	pinCodeInput0 := page.Locator("#pinCode0")
	otpInput := page.Locator("[automation-id=otp-input]")
	submitBtn := page.Locator("[automation-id=button-submit]")
	passwordInput := page.Locator("[automation-id=password-input]")

	log.Println("\033[1A\033[K 9/9 🔸 Login steps handled")

	if visible, _ := phoneInput.IsVisible(); visible {

		// 1) Вводим телефон
		log.Println(" 1/9 🌀 Entering phone...")

		if err := phoneInput.Fill(phoneMasked); err != nil {
			return "", fmt.Errorf("Fill: %v", err)
		}
		log.Println("\033[1A\033[K 1/9 🔹 Phone entered")

		log.Println(" 2/9 🌀 Submitting phone...")

		if err := submitBtn.Click(); err != nil {
			return "", fmt.Errorf("Click submit: %v", err)
		}

		log.Println("\033[1A\033[K 2/9 🔹 Phone submitted")

		// Подстраховка — ждём networkidle, но полагаемся на селекторы:
		waitPage(page)

		log.Println(" 3/9 🌀 Handling OTP and PIN...")
		// 2) Ждём появления поля для кода из СМС и просим тебя ввести код
		if err := otpInput.WaitFor(); err != nil {
			return "", fmt.Errorf("WaitFor otp-input: %v", err)
		}
		log.Println("\033[1A\033[K 3/9 🔹 OTP and PIN handled")


		log.Println(" 4/9 🌀 Fetching SMS code...")

		time.Sleep(time.Minute)
		code, err := sms.FetchCode("nX45hGyTIdy076742pdVvpfCdBp4wTFYk9mJ5HSXiKCPQtgKRcezloGKuroLVoGn", "topdrop.fun", "/core/ajax.php")
		if err != nil {
			return "", fmt.Errorf("sms.FetchCode: %v", err)
		}
		log.Println("\033[1A\033[K 4/9 🔹 SMS code read")

		log.Println(" 5/9 🌀 Filling OTP code...")
		if err := otpInput.Fill(code); err != nil {
			return "", fmt.Errorf("Fill otp code: %v", err)
		}
		log.Println("\033[1A\033[K 5/9 🔹 OTP code filled")

		// 3) После ввода кода может открыться экран с пин-кодом
		waitPage(page)

		err = passwordInput.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(3000),
			State:   playwright.WaitForSelectorStateVisible,
		})
		if err == nil {
			log.Println(" 6/9 🌀 Setting password...")
			if err := passwordInput.Fill(password); err != nil {
				return "", fmt.Errorf("Fill password: %v", err)
			}
			log.Println("\033[1A\033[K 6/9 🔹 Password set")

			// Подтверждаем
			log.Println(" 7/9 🌀 Submitting...")
			if err := submitBtn.Click(); err != nil {
				return "", fmt.Errorf("Click submit (after password): %v", err)
			}
			log.Println("\033[1A\033[K 7/9 🔹 Submitted")
			waitPage(page)
		}

		log.Println(" 8/9 🌀 Setting PIN code...")
		if err := setPinCode(page, pinCode); err != nil {
			return "", fmt.Errorf("setPinCode: %v", err)
		}
		log.Println("\033[1A\033[K 8/9 🔹 PIN code set")

		// 4) Подтверждаем
		log.Println(" 9/9 🌀 Submitting...")
		if err := submitBtn.Click(); err != nil {
			return "", fmt.Errorf("submitBtn.Click: %v", err)
		}
		log.Println("\033[1A\033[K 9/9 🔹 Submitted")
	} else if visible, _ := pinCodeInput0.IsVisible(); visible {
		// Телефон не спрашивали — возможно, сразу пин-код
		log.Println(" 1/1 🌀 Setting PIN code (no phone step)...")
		if err := setPinCode(page, pinCode); err != nil {
			return "", fmt.Errorf("setPinCode (no phone step): %v", err)
		}
		log.Println("\033[1A\033[K 1/1 🔹 PIN code set (no phone step)")
	}

	waitState = playwright.WaitUntilState("networkidle") // допустимое значение WaitUntil
	_, err = page.Goto("https://www.tbank.ru/mybank/operations", playwright.PageGotoOptions{
		WaitUntil: &waitState, // см. предупреждение в доках — networkidle не для всех случаев
	})

	// Небольшая пауза — дождаться завершения редиректов/сетевых запросов
	log.Println(" 1/3 🌀 Waiting for redirects...")
	waitPage(page)

	log.Println("\033[1A\033[K 1/3 ✅ Redirects done")

	log.Println(" 2/3 🌀 Fetching cookies...")

	cookies, err := ctx.Cookies()
	log.Println("\033[1A\033[K 2/3 ✅ Cookies fetched")
	if err != nil {
		return "", fmt.Errorf("cookies: %v", err)
	}

	log.Println(" 3/3 🌀 Cookies handled...")

	for _, c := range cookies {
		if c.Name == "psid" {
			log.Println("\033[1A\033[K 3/3 ✅ Cookies found")
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

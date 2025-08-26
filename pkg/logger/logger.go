package logger

import (
	"io"
	"log"
	"os"
)

var (
	logger *log.Logger
)

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
)

func Init() error {
	file, err := os.OpenFile("../logs/payments.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	multiWriter := io.MultiWriter(os.Stdout, file)

	logger = log.New(multiWriter, "", log.Ldate|log.Ltime|log.Lshortfile)

	return nil

	/* details := fmt.Sprintf("SUM: %s DATE: %d", sum.StringFixed(2), paidAt) */

	/*
		 	switch payment.Status {
			case StatusMade:
				log.Println(green + payment.Text + reset)
				logger.Println("[INFO]", payment.Text, details)
			case StatusPaid:
				log.Println(yellow + payment.Text + reset)
				logger.Println("[WARN]", payment.Text, details)
			case StatusError:
				log.Println(red + payment.Text + reset)
				logger.Println("[ERROR]", payment.Text, details)
			default:
				log.Println(red + "Unknown error" + reset)
				logger.Println("[ERROR] Unknown error")
			}
	*/
}

func Warnln(v ...any) {
	args := append([]any{"[WARN]"}, v...)
	logger.Println(args...)
}

func Warnf(format string, v ...any) {
	args := append([]any{"[WARN]"}, v...)
	logger.Printf(format, args...)
}

func Warn(v ...any) {
	args := append([]any{"[WARN]"}, v...)
	logger.Print(args...)
}

func Errorln(v ...any) {
	args := append([]any{"[ERROR]"}, v...)
	logger.Println(args...)
}

func Errorf(format string, v ...any) {
	args := append([]any{"[ERROR]"}, v...)
	logger.Printf(format, args...)
}

func Error(v ...any) {
	args := append([]any{"[ERROR]"}, v...)
	logger.Print(args...)
}

func Infoln(v ...any) {
	args := append([]any{"[INFO]"}, v...)
	logger.Println(args...)
}

func Infof(format string, v ...any) {
	args := append([]any{"[INFO]"}, v...)
	logger.Printf(format, args...)
}

func Info(v ...any) {
	args := append([]any{"[INFO]"}, v...)
	logger.Print(args...)
}
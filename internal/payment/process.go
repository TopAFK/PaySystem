package payment

import (
	"database/sql"

	"github.com/shopspring/decimal"
)

type Status string

const (
	StatusSucceeded    Status = "succeeded"
	StatusDuplicate    Status = "duplicate"
	StatusManyPayments Status = "many_payments"
	StatusError        Status = "error"
)

type Payment struct {
	ID     string `json:"id"`
	Status uint8  `json:"status"`
}

func Process(db *sql.DB, paidAt int64, sum decimal.Decimal) (Status, error) {
	rows, err := db.Query("SELECT `id`, `status` FROM `payments` WHERE `sum` = ? AND (created_at < FROM_UNIXTIME(?) AND created_at > FROM_UNIXTIME(?))", sum, paidAt, paidAt-1800)

	if err != nil {
		return "", err
	}
	defer rows.Close()
	var payments []Payment

	for rows.Next() {
		var payment Payment
		if err := rows.Scan(&payment.ID, &payment.Status); err != nil {
			return "", err
		}
		payments = append(payments, payment)
	}
	if rows.Err() != nil {
		return "", rows.Err()
	}
	
	for _, p := range payments {
		if p.Status == 1 {
			return StatusDuplicate, nil
		}
	}

	return StatusSucceeded, nil
}

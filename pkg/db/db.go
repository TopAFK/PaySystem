package db

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

var TD *sql.DB

func Init(DSN string) error {
	var err error
	TD, err = sql.Open("mysql", DSN)
	if err != nil {
		return err
	}
	TD.SetConnMaxLifetime(0)
	TD.SetMaxOpenConns(10)
	TD.SetMaxIdleConns(10)

	if err := TD.Ping(); err != nil {
		return err
	}
	
	return nil
}
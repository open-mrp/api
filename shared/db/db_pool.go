package db

import (
	"database/sql"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func NewDbPool(dbURL string) (*sql.DB, error) {
	if dbURL != "" {
		params := []string{}

		if !strings.Contains(dbURL, "parseTime=") {
			params = append(params, "parseTime=true")
		}
		if !strings.Contains(dbURL, "loc=") {
			params = append(params, "loc=UTC")
		}
		if !strings.Contains(dbURL, "time_zone=") {
			params = append(params, "time_zone=%27%2B00%3A00%27")
		}

		if len(params) > 0 {
			paramString := strings.Join(params, "&")
			if strings.Contains(dbURL, "?") {
				dbURL += "&" + paramString
			} else {
				dbURL += "?" + paramString
			}
		}
	}

	db, err := sql.Open("mysql", dbURL)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(50)

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

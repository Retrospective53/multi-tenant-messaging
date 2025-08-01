package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

const (
	dbDriver = "postgres"
	dbSource = "postgresql://user:pass@localhost:5432/app?sslmode=disable"
)

var DB *sql.DB

func InitDB(dsn string) error {
	var err error
	DB, err = sql.Open(dbDriver, dbSource)
	if err != nil {
		log.Fatal("cannot connect to the database")
	}

	return err
}

package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

type Config struct {
	Server   string
	Port     int
	User     string
	Password string
	Database string
}

func Open(cfg Config) (*sql.DB, error) {
	connString := fmt.Sprintf(
		"server=%s;user id=%s;password=%s;port=%d;database=%s;encrypt=disable",
		cfg.Server, cfg.User, cfg.Password, cfg.Port, cfg.Database,
	)
	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

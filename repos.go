package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"
)

type InMemoryUrlRepoRecord struct {
	id      [idByteLen]byte
	longUrl string
}

// Not efficient. Just for testing.

type InMemoryURLRepo struct { // For use in testing, not robust at all
	records []InMemoryUrlRepoRecord
}

func (imur *InMemoryURLRepo) GetId(longUrl string) ([idByteLen]byte, error) {
	for _, record := range imur.records {
		if record.longUrl == longUrl {
			return record.id, nil
		}
	}
	return [idByteLen]byte{}, nil
}

func (imur *InMemoryURLRepo) GetLongURL(id [idByteLen]byte) (string, error) {
	for _, record := range imur.records {
		if record.id == id {
			return record.longUrl, nil
		}
	}
	return "", nil
}

func (imur *InMemoryURLRepo) StoreURLRecord(id [idByteLen]byte, longUrl string) error {
	imur.records = append(imur.records, InMemoryUrlRepoRecord{id, longUrl})
	return nil
}

type SQLRepo struct {
	db         *sql.DB
	getIdStmt  *sql.Stmt
	insertStmt *sql.Stmt
}

func buildSQLRepo(driver string, dataSourceName string) *SQLRepo {
	db, err := sql.Open(driver, dataSourceName)
	if err != nil {
		log.Fatal("unable to open database", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ping the database using exponential backoff
	s := 1
	for {
		err = db.PingContext(ctx)
		if err == nil {
			break
		} else {
			if s == 32 {
				log.Fatal("unable to ping database", err)
			}
			time.Sleep(time.Duration(s) * time.Second)
			s *= 2
		}
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	insertStmt, err := db.PrepareContext(context.Background(), "INSERT INTO urls (id, long_url) VALUES (?, ?)")
	if err != nil {
		log.Fatal("unable to prepare insert statement", err)
	}

	getIdStmt, err := db.PrepareContext(context.Background(), "SELECT id FROM urls WHERE long_url = ?")
	if err != nil {
		log.Fatal("unable to prepare get id statement", err)
	}

	return &SQLRepo{db, getIdStmt, insertStmt}
}

func (sr *SQLRepo) GetId(longUrl string) ([idByteLen]byte, error) {
	var id [idByteLen]byte
	err := sr.getIdStmt.QueryRowContext(context.Background(), longUrl).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return [idByteLen]byte{}, nil // No id exists
		}
		return [idByteLen]byte{}, err // An error occurred
	}
	return id, nil // Successfully found
}

func (sr *SQLRepo) GetLongURL(id [idByteLen]byte) (string, error) {
	var longUrl string
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := sr.db.QueryRowContext(ctx, "SELECT long_url FROM urls WHERE id = ?", id).Scan(&longUrl)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil // No long URL exists
		}
		return "", err // An error occurred
	}
	return longUrl, nil // Successfully found
}

func (sr *SQLRepo) StoreURLRecord(id [idByteLen]byte, longUrl string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := sr.insertStmt.ExecContext(ctx, id[:], longUrl)
	return err
}

package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"
)

type InMemoryUrlRepoRecord struct {
	id       [idByteLen]byte
	longUrl  string
	shortUrl string
}

// Not efficient. Just for testing.

type InMemoryURLRepo struct { // For use in testing, not robust at all
	records []InMemoryUrlRepoRecord
}

func (imur *InMemoryURLRepo) GetShortURL(longUrl string) (string, error) {
	for _, record := range imur.records {
		if record.longUrl == longUrl {
			return record.shortUrl, nil
		}
	}
	return "", nil
}

func (imur *InMemoryURLRepo) GetLongURL(shortUrl string) (string, error) {
	for _, record := range imur.records {
		if record.shortUrl == shortUrl {
			return record.longUrl, nil
		}
	}
	return "", nil
}

func (imur *InMemoryURLRepo) StoreURLRecord(id [idByteLen]byte, longUrl string, shortUrl string) error {
	imur.records = append(imur.records, InMemoryUrlRepoRecord{id, longUrl, shortUrl})
	return nil
}

type SQLRepo struct {
	db         *sql.DB
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

	insertStmt, err := db.PrepareContext(context.Background(), "INSERT INTO urls (id, long_url, short_url) VALUES (?, ?, ?)")
	if err != nil {
		log.Fatal("unable to prepare insert statement", err)
	}

	return &SQLRepo{db, insertStmt}
}

func (sr *SQLRepo) GetShortURL(longUrl string) (string, error) {
	var shortUrl string
	//ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	//defer cancel()
	err := sr.db.QueryRowContext(context.Background(), "SELECT short_url FROM urls WHERE long_url = ?", longUrl).Scan(&shortUrl)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil // No short URL exists
		}
		return "", err // An error occurred
	}
	return shortUrl, nil // Successfully found
}

func (sr *SQLRepo) GetLongURL(shortUrl string) (string, error) {
	var longUrl string
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := sr.db.QueryRowContext(ctx, "SELECT long_url FROM urls WHERE short_url = ?", shortUrl).Scan(&longUrl)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil // No long URL exists
		}
		return "", err // An error occurred
	}
	return longUrl, nil // Successfully found
}

func (sr *SQLRepo) StoreURLRecord(id [idByteLen]byte, longUrl string, shortUrl string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := sr.insertStmt.ExecContext(ctx, id[:], longUrl, shortUrl)
	return err
}

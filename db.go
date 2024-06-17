package main

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type UrlDB interface {
	GetId(longUrl string) (UrlId, error) // Zeroed out if not found
	GetLongURL(id UrlId) (string, error) // Empty string if not found
	StoreURLRecord(id UrlId, longUrl string) error
	Connected() bool
}

type InMemoryUrlDbRecord struct {
	id      UrlId
	longUrl string
}

type InMemoryUrlDb struct { // For use in testing, not robust at all
	records []InMemoryUrlDbRecord
}

func (imur *InMemoryUrlDb) GetId(longUrl string) (UrlId, error) {
	for _, record := range imur.records {
		if record.longUrl == longUrl {
			return record.id, nil
		}
	}
	return UrlId{}, nil
}

func (imur *InMemoryUrlDb) GetLongURL(id UrlId) (string, error) {
	for _, record := range imur.records {
		if record.id == id {
			return record.longUrl, nil
		}
	}
	return "", nil
}

func (imur *InMemoryUrlDb) StoreURLRecord(id UrlId, longUrl string) error {
	imur.records = append(imur.records, InMemoryUrlDbRecord{id, longUrl})
	return nil
}

func (imur *InMemoryUrlDb) Connected() bool {
	return true
}

type MySQLUrlDB struct {
	db         *sql.DB
	getIdStmt  *sql.Stmt
	insertStmt *sql.Stmt
}

func buildSQLRepo(driver string, dataSourceName string) (*MySQLUrlDB, func(), error) {
	// Configure DB
	db, err := sql.Open(driver, dataSourceName)
	if err != nil {
		return nil, nil, err
	}

	sr := MySQLUrlDB{db: db}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	if !sr.Connected() {
		return nil, nil, errors.New("Failed to connect to database")
	}

	// Prepare statements
	insertStmt, err := db.PrepareContext(context.Background(), "INSERT INTO urls (id, long_url) VALUES (?, ?)")
	if err != nil {
		return nil, nil, err
	}

	getIdStmt, err := db.PrepareContext(context.Background(), "SELECT id FROM urls WHERE long_url = ?")
	if err != nil {
		return nil, nil, err
	}

	sr.insertStmt = insertStmt
	sr.getIdStmt = getIdStmt

	// Create cleanup closure
	dbTidy := func() {
		insertStmt.Close()
		getIdStmt.Close()
		db.Close()
	}

	return &sr, dbTidy, nil
}

func (sr *MySQLUrlDB) GetId(longUrl string) (UrlId, error) {
	var idSlice []byte
	var id UrlId
	err := sr.getIdStmt.QueryRowContext(context.Background(), longUrl).Scan(&idSlice)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UrlId{}, nil // No id exists
		}
		return UrlId{}, err // An error occurred
	}
	copy(id[:], idSlice)
	return id, nil // Successfully found
}

func (sr *MySQLUrlDB) GetLongURL(id UrlId) (string, error) {
	var longUrl string
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := sr.db.QueryRowContext(ctx, "SELECT long_url FROM urls WHERE id = ?", id[:]).Scan(&longUrl)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil // No long URL exists
		}
		return "", err // An error occurred
	}
	return longUrl, nil // Successfully found
}

func (sr *MySQLUrlDB) StoreURLRecord(id UrlId, longUrl string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := sr.insertStmt.ExecContext(ctx, id[:], longUrl)
	return err
}

func (sr *MySQLUrlDB) Connected() bool {
	timeout := 6250 * time.Microsecond // Expands to ~30s with 10 attempts
	attempts := 0
	limit := 10
	for {
		if attempts == limit {
			return false
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		err := sr.db.PingContext(ctx)

		// Connected successfully
		if err == nil {
			cancel()
			return true
		}

		// Try again w/ exponential backoff
		attempts++
		timeout *= 2
		<-ctx.Done() // Wait out the timeout
		cancel()
	}
}

package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"os"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run() error {
	db, dbTidy, err := buildSQLRepo("mysql", "root@tcp(db:3306)/urlshortener")
	if err != nil {
		return fmt.Errorf("failed to build SQL db: %w", err)
	}
	defer dbTidy()

	app := &URLShortenerApp{
		urlRepo:     db,
		idGenerator: newUniqueIDGenerator(),
	}

	s, err := newServer(gin.Default(), app, db)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Only handle if err != http.ErrServerClosed
	if err = http.ListenAndServe(":"+port, s); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

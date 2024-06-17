package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type server struct {
	routes *gin.Engine
	app    *URLShortenerApp
	db     UrlDB
}

func newServer(r *gin.Engine, app *URLShortenerApp, db UrlDB) (*server, error) {
	s := &server{
		routes: r,
		app:    app,
		db:     db,
	}
	s.addRoutes()
	return s, nil
}

func (s *server) addRoutes() {
	s.routes.GET("api/v1/health", s.handleHealth())
	s.routes.POST("api/v1/shorten", s.handleShorten())
	s.routes.GET("api/v1/redirect", s.handleRedirect())
}

func (s *server) handleHealth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.db.Connected() {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func (s *server) handleShorten() gin.HandlerFunc {
	return func(c *gin.Context) {
		longUrl := c.Query("longUrl")
		if longUrl == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "param `longUrl` is required"})
			return
		}
		shortUrl, err := s.app.shorten(longUrl)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"shortUrl": shortUrl})
	}
}

func (s *server) handleRedirect() gin.HandlerFunc {
	return func(c *gin.Context) {
		shortUrl := c.Query("shortUrl")
		if shortUrl == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "param `shortUrl` is required"})
			return
		}
		longUrl, err := s.app.redirect(shortUrl)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		if longUrl == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "shortUrl not known"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"longUrl": longUrl})
	}
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.routes.ServeHTTP(w, r)
}

package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const idByteLen = 7
const shortURLLen = 10
const base62Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func encodeBase62(id [idByteLen]byte) string {
	// Each word is 11 bits, drop bit in the 33rd place (which is zero)
	firstWord := (uint16(id[0]) << 3) | (uint16(id[1] >> 5))
	secondWord := (uint16(id[1]&0x1f) << 6) | uint16(id[2]>>2)
	thirdWord := (uint16(id[2]&0x3) << 9) | (uint16(id[3]) << 1) | (uint16(id[4]>>6) & 0x1) // Zeros padded bit
	fourthWord := (uint16(id[4]&0x3f) << 5) | uint16(id[5]>>3)
	fifthWord := (uint16(id[5]&0x7) << 8) | uint16(id[6])

	shortUrl := make([]byte, shortURLLen)

	shortUrl[0] = base62Chars[firstWord/62]
	shortUrl[1] = base62Chars[firstWord%62]
	shortUrl[2] = base62Chars[secondWord/62]
	shortUrl[3] = base62Chars[secondWord%62]
	shortUrl[4] = base62Chars[thirdWord/62]
	shortUrl[5] = base62Chars[thirdWord%62]
	shortUrl[6] = base62Chars[fourthWord/62]
	shortUrl[7] = base62Chars[fourthWord%62]
	shortUrl[8] = base62Chars[fifthWord/62]
	shortUrl[9] = base62Chars[fifthWord%62]

	return string(shortUrl)
}

type URLRepo interface {
	GetShortURL(longUrl string) (string, error)
	GetLongURL(shortUrl string) (string, error)
	StoreURLRecord(id [idByteLen]byte, longUrl string, shortUrl string) error
}

type UniqueIDGenerator interface {
	GenerateUniqueID() [idByteLen]byte
}

type UniqueIDGeneratorImpl struct {
	seq  uint32
	lock sync.Mutex
	cond *sync.Cond
}

func NewUniqueIDGenerator() *UniqueIDGeneratorImpl {
	u := &UniqueIDGeneratorImpl{}
	u.cond = sync.NewCond(&u.lock)
	u.startSeqReset()
	return u
}

func (uidg *UniqueIDGeneratorImpl) GenerateUniqueID() [idByteLen]byte {
	uidg.lock.Lock()
	defer uidg.lock.Unlock()

	// Wait while seq is >2^23
	for uidg.seq >= (1 << 24) {
		uidg.cond.Wait()
	}

	seconds := uint32(time.Now().Unix())
	id := [idByteLen]byte{}
	err := encodeID(&id, seconds, uidg.seq)
	if err != nil {
		panic(err)
	}
	uidg.seq++
	return id
}

func encodeID(buf *[idByteLen]byte, seconds uint32, seq uint32) error {
	if seq > 8388607 {
		return fmt.Errorf("seq is too large, should be less than 2^23")
	}

	buf[0] = byte(seconds >> 24)
	buf[1] = byte(seconds >> 16)
	buf[2] = byte(seconds >> 8)
	buf[3] = byte(seconds)
	buf[4] = byte(seq >> 16) // Padding implied by condition above
	buf[5] = byte(seq >> 8)
	buf[6] = byte(seq)

	return nil
}

func (uidg *UniqueIDGeneratorImpl) startSeqReset() {
	ticker := time.NewTicker(time.Second)
	go func() {
		for _ = range ticker.C {
			uidg.lock.Lock()
			uidg.seq = 0
			uidg.cond.Broadcast()
			uidg.lock.Unlock()
		}
	}()
}

type URLShortenerApp struct {
	urlRepo     URLRepo
	idGenerator UniqueIDGenerator
}

func (app *URLShortenerApp) shorten(longUrl string) (string, error) {
	var shortUrl string
	var err error

	// See if shortUrl already exists
	shortUrl, err = app.urlRepo.GetShortURL(longUrl)
	if err != nil {
		return "", err
	}

	// If not, generate and save
	if shortUrl == "" { // Generate short url, reassign
		id := app.idGenerator.GenerateUniqueID()
		shortUrl = encodeBase62(id)
		err = app.urlRepo.StoreURLRecord(id, longUrl, shortUrl)
		if err != nil {
			return "", err
		}
	}

	// return
	return shortUrl, nil
}

func (app *URLShortenerApp) redirect(shortUrl string) (string, error) {
	longUrl, err := app.urlRepo.GetLongURL(shortUrl)
	if err != nil {
		return "", err
	}
	return longUrl, nil
}

func main() {
	idg := NewUniqueIDGenerator()

	dsn := "root@tcp(db:3306)/urlshortener"
	//dsn := "root@tcp(localhost:3306)/urlshortener"
	repo := buildSQLRepo("mysql", dsn)

	//repo := &InMemoryURLRepo{make([]InMemoryUrlRepoRecord, 100000)}

	app := &URLShortenerApp{
		urlRepo:     repo,
		idGenerator: idg,
	}

	r := gin.Default()

	r.GET("/api/v1/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	r.POST("api/v1/shorten", func(c *gin.Context) {
		longUrl := c.Query("longUrl")
		if longUrl == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "param `longUrl` is required"})
			return
		}
		shortUrl, err := app.shorten(longUrl)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"shortUrl": shortUrl})
	})

	r.GET("api/v1/shortUrl", func(c *gin.Context) {
		shortUrl := c.Query("shortUrl")
		if shortUrl == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "param `shortUrl` is required"})
			return
		}
		longUrl, err := app.redirect(shortUrl)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		if longUrl == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "shortUrl not known"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"longUrl": longUrl})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.Run(":" + port)
}

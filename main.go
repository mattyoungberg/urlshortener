package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const idByteLen = 7
const shortURLLen = 10
const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz" // ascending order

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

func decodeBase62(shortUrl string) [idByteLen]byte {
	firstWord := strings.Index(base62Chars, string(shortUrl[0]))*62 + strings.Index(base62Chars, string(shortUrl[1]))
	secondWord := strings.Index(base62Chars, string(shortUrl[2]))*62 + strings.Index(base62Chars, string(shortUrl[3]))
	thirdWord := strings.Index(base62Chars, string(shortUrl[4]))*62 + strings.Index(base62Chars, string(shortUrl[5]))
	fourthWord := strings.Index(base62Chars, string(shortUrl[6]))*62 + strings.Index(base62Chars, string(shortUrl[7]))
	fifthWord := strings.Index(base62Chars, string(shortUrl[8]))*62 + strings.Index(base62Chars, string(shortUrl[9]))

	id := [idByteLen]byte{}

	id[0] = byte(firstWord >> 3)
	id[1] = byte((firstWord&0x7)<<5) | byte(secondWord>>6)
	id[2] = byte((secondWord&0x3f)<<2) | byte(thirdWord>>9)
	id[3] = byte(thirdWord >> 1)
	id[4] = byte((thirdWord&0x1)<<6) | byte(fourthWord>>5)
	id[5] = byte((fourthWord&0x1f)<<3) | byte(fifthWord>>8)
	id[6] = byte(fifthWord)

	return id
}

type URLRepo interface {
	GetId(longUrl string) ([idByteLen]byte, error) // Zeroed out if not found
	GetLongURL(id [idByteLen]byte) (string, error) // Empty string if not found
	StoreURLRecord(id [idByteLen]byte, longUrl string) error
}

type UniqueIDGenerator interface {
	GenerateUniqueID() [idByteLen]byte
}

type UniqueIDGeneratorImpl struct {
	seconds uint32
	seq     uint32
	lock    sync.Mutex
	cond    *sync.Cond
}

func NewUniqueIDGenerator() *UniqueIDGeneratorImpl {
	u := &UniqueIDGeneratorImpl{}
	u.seconds = uint32(time.Now().Unix())
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

	id := [idByteLen]byte{}
	err := encodeID(&id, uidg.seconds, uidg.seq)
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
			uidg.seconds = uint32(time.Now().Unix())
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
	var id [idByteLen]byte
	var err error

	// See if shortUrl already exists
	id, err = app.urlRepo.GetId(longUrl)
	if err != nil {
		return "", err
	}

	// If not, generate and save
	if id == [idByteLen]byte{} { // Generate short url, reassign
		id = app.idGenerator.GenerateUniqueID()
		err = app.urlRepo.StoreURLRecord(id, longUrl)
		if err != nil {
			return "", err
		}
	}

	// return
	return encodeBase62(id), nil
}

func (app *URLShortenerApp) redirect(shortUrl string) (string, error) {
	id := decodeBase62(shortUrl)
	longUrl, err := app.urlRepo.GetLongURL(id)
	if err != nil {
		return "", err
	}
	return longUrl, nil
}

func main() {
	idg := NewUniqueIDGenerator()

	dsn := "root@tcp(db:3306)/urlshortener"
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

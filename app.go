package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const idByteLen = 7
const shortURLLen = 10
const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz" // ascending order

type UrlId [idByteLen]byte

func encodeBase62(id UrlId) string {
	// Each word is 11 bits, drop the bit in the 33rd place (which is zero)
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

func decodeBase62(shortUrl string) UrlId {
	firstWord := strings.Index(base62Chars, string(shortUrl[0]))*62 + strings.Index(base62Chars, string(shortUrl[1]))
	secondWord := strings.Index(base62Chars, string(shortUrl[2]))*62 + strings.Index(base62Chars, string(shortUrl[3]))
	thirdWord := strings.Index(base62Chars, string(shortUrl[4]))*62 + strings.Index(base62Chars, string(shortUrl[5]))
	fourthWord := strings.Index(base62Chars, string(shortUrl[6]))*62 + strings.Index(base62Chars, string(shortUrl[7]))
	fifthWord := strings.Index(base62Chars, string(shortUrl[8]))*62 + strings.Index(base62Chars, string(shortUrl[9]))

	id := UrlId{}

	id[0] = byte(firstWord >> 3)
	id[1] = byte((firstWord&0x7)<<5) | byte(secondWord>>6)
	id[2] = byte((secondWord&0x3f)<<2) | byte(thirdWord>>9)
	id[3] = byte(thirdWord >> 1)
	id[4] = byte((thirdWord&0x1)<<6) | byte(fourthWord>>5)
	id[5] = byte((fourthWord&0x1f)<<3) | byte(fifthWord>>8)
	id[6] = byte(fifthWord)

	return id
}

type URLShortenerApp struct {
	urlRepo     UrlDB
	idGenerator UniqueIDGenerator
}

func (app *URLShortenerApp) shorten(longUrl string) (string, error) {
	var id UrlId
	var err error

	// See if shortUrl already exists
	id, err = app.urlRepo.GetId(longUrl)
	if err != nil {
		return "", err
	}

	// If not, generate and save
	if (id == UrlId{}) { // Generate short url, reassign
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

type UniqueIDGenerator interface {
	GenerateUniqueID() UrlId
}

type UniqueIDGeneratorImpl struct {
	seconds uint32
	seq     uint32
	lock    sync.Mutex
	cond    *sync.Cond
}

func newUniqueIDGenerator() *UniqueIDGeneratorImpl {
	u := &UniqueIDGeneratorImpl{}
	u.seconds = uint32(time.Now().Unix())
	u.cond = sync.NewCond(&u.lock)
	u.startSeqReset()
	return u
}

func (uidg *UniqueIDGeneratorImpl) GenerateUniqueID() UrlId {
	uidg.lock.Lock()
	defer uidg.lock.Unlock()

	// Wait while seq is >2^23
	for uidg.seq >= (1 << 24) {
		uidg.cond.Wait()
	}

	id := UrlId{}
	err := encodeID(&id, uidg.seconds, uidg.seq)
	if err != nil {
		panic(err)
	}
	uidg.seq++
	return id
}

func (uidg *UniqueIDGeneratorImpl) startSeqReset() {
	ticker := time.NewTicker(time.Second)
	go func() {
		for range ticker.C {
			uidg.lock.Lock()
			uidg.seconds = uint32(time.Now().Unix())
			uidg.seq = 0
			uidg.cond.Broadcast()
			uidg.lock.Unlock()
		}
	}()
}

func encodeID(buf *UrlId, seconds uint32, seq uint32) error {
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

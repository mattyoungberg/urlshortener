package main

import (
	"math"
	"testing"
	"time"
)

func TestEncodeBase62_Zeroes(t *testing.T) {
	var idArr [idByteLen]byte

	for i := 0; i < idByteLen; i++ {
		idArr[i] = 0x00
	}

	encoded := encodeBase62(idArr)

	for _, char := range encoded {
		if char != '0' { // Implies that base62 algo encodes 0x0 w/ A instead of 0
			t.Error("Encoded short URL should be all 'A's")
		}
	}

	if len(encoded) != shortURLLen {
		t.Error("Encoded short URL should be of the correct length")
	}
}

func TestDecodeBase62_Zeroes(t *testing.T) {
	var idArr [idByteLen]byte

	for i := 0; i < idByteLen; i++ {
		idArr[i] = 0x00
	}

	encoded := encodeBase62(idArr)

	decoded := decodeBase62(encoded)

	for i := 0; i < idByteLen; i++ {
		if decoded[i] != idArr[i] {
			t.Error("Decoded id should be the same as the original id")
		}
	}
}

func TestEncodeBase62_Max(t *testing.T) {
	var idArr [idByteLen]byte
	expected := "X1X1X1X1X1" // 2047 for each 11-bit word, or 2^11 - 1

	for i := 0; i < idByteLen; i++ {
		idArr[i] = 0xff
	}

	// Mask the padding bit
	idArr[4] = idArr[4] & 0b01111111

	actual := encodeBase62(idArr)

	if expected != actual {
		t.Fail()
	}
}

func TestDecodeBase62_Max(t *testing.T) {
	var idArr [idByteLen]byte

	for i := 0; i < idByteLen; i++ {
		idArr[i] = 0xff
	}

	// Mask the padding bit
	idArr[4] = idArr[4] & 0b01111111

	encoded := encodeBase62(idArr)
	decoded := decodeBase62(encoded)

	for i := 0; i < idByteLen; i++ {
		if decoded[i] != idArr[i] {
			t.Error("Decoded id should be the same as the original id")
		}
	}
}

func TestEncodeBase62_ManualCalc1(t *testing.T) {
	num := 0b01110010001011011101100010010110001011111111010001010001
	expected := "EjEI4qOkHp"
	id := [idByteLen]byte{}

	id[0] = byte(num >> 48)
	id[1] = byte(num >> 40)
	id[2] = byte(num >> 32)
	id[3] = byte(num >> 24)
	id[4] = byte(num >> 16)
	id[5] = byte(num >> 8)
	id[6] = byte(num)

	actual := encodeBase62(id)

	if expected != actual {
		t.Fail()
	}
}

func TestDecodeBase62_ManualCalc1(t *testing.T) {
	num := 0b01110010001011011101100010010110001011111111010001010001
	id := [idByteLen]byte{}

	id[0] = byte(num >> 48)
	id[1] = byte(num >> 40)
	id[2] = byte(num >> 32)
	id[3] = byte(num >> 24)
	id[4] = byte(num >> 16)
	id[5] = byte(num >> 8)
	id[6] = byte(num)

	actual := encodeBase62(id)
	decoded := decodeBase62(actual)

	for i := 0; i < idByteLen; i++ {
		if decoded[i] != id[i] {
			t.Error("Decoded id should be the same as the original id")
		}
	}
}

func TestEncodeBase62_ManualCalc2(t *testing.T) {
	num := 0b01010011100001010011010100000000010111011111100111100110
	expected := "Am5N8HFT7q"
	id := [idByteLen]byte{}

	id[0] = byte(num >> 48)
	id[1] = byte(num >> 40)
	id[2] = byte(num >> 32)
	id[3] = byte(num >> 24)
	id[4] = byte(num >> 16)
	id[5] = byte(num >> 8)
	id[6] = byte(num)

	actual := encodeBase62(id)

	if expected != actual {
		t.Fail()
	}
}

func TestDecodeBase62_ManualCalc2(t *testing.T) {
	num := 0b01010011100001010011010100000000010111011111100111100110
	id := [idByteLen]byte{}

	id[0] = byte(num >> 48)
	id[1] = byte(num >> 40)
	id[2] = byte(num >> 32)
	id[3] = byte(num >> 24)
	id[4] = byte(num >> 16)
	id[5] = byte(num >> 8)
	id[6] = byte(num)

	actual := encodeBase62(id)
	decoded := decodeBase62(actual)

	for i := 0; i < idByteLen; i++ {
		if decoded[i] != id[i] {
			t.Error("Decoded id should be the same as the original id")
		}
	}
}

func TestEncodeBase62_ManualCalc3(t *testing.T) {
	num := 0b01101111010110001101110110100111011110001101000011001000
	expected := "EMPfDfTK3E"
	id := [idByteLen]byte{}

	id[0] = byte(num >> 48)
	id[1] = byte(num >> 40)
	id[2] = byte(num >> 32)
	id[3] = byte(num >> 24)
	id[4] = byte(num >> 16)
	id[5] = byte(num >> 8)
	id[6] = byte(num)

	actual := encodeBase62(id)

	if expected != actual {
		t.Fail()
	}
}

func TestDecodeBase62_ManualCalc3(t *testing.T) {
	num := 0b01101111010110001101110110100111011110001101000011001000
	id := [idByteLen]byte{}

	id[0] = byte(num >> 48)
	id[1] = byte(num >> 40)
	id[2] = byte(num >> 32)
	id[3] = byte(num >> 24)
	id[4] = byte(num >> 16)
	id[5] = byte(num >> 8)
	id[6] = byte(num)

	actual := encodeBase62(id)
	decoded := decodeBase62(actual)

	for i := 0; i < idByteLen; i++ {
		if decoded[i] != id[i] {
			t.Error("Decoded id should be the same as the original id")
		}
	}
}

func TestUniqueIDGeneratorImpl_GenerateUniqueID(t *testing.T) {
	uidg := NewUniqueIDGenerator()

	id1 := uidg.GenerateUniqueID()
	id2 := uidg.GenerateUniqueID()

	if id1 == id2 {
		// Incredibly small chance of failure, more of a sanity check that we're not
		// accidentally generating the same id twice.
		t.Error("Generator produced two identical ids")
	}
}

func TestUniqueIDGeneratorImpl_GenerateUniqueID_GuaranteeAllUniqueIn1Second(t *testing.T) {
	set := make(map[[idByteLen]byte]bool)
	uidg := NewUniqueIDGenerator()

	done := make(chan bool)
	go func() {
		time.Sleep(time.Second)
		done <- true
	}()

	for {
		select {
		case <-done:
			t.Logf("Generated %d unique ids in the test loop", len(set))
			return
		default:
			id := uidg.GenerateUniqueID()
			if _, exists := set[id]; exists {
				t.Errorf("Generator produced the same id twice in a second")
			}
			set[id] = true
		}
	}
}

func TestUniqueIDGeneratorImpl_GenerateUniqueID_Guarantee100000Unique(t *testing.T) {
	set := make(map[[idByteLen]byte]bool)
	uidg := NewUniqueIDGenerator()

	for i := 0; i < 100000; i++ {
		id := uidg.GenerateUniqueID()
		if _, exists := set[id]; exists {
			t.Errorf("Generator produced the same id twice in 100000 iterations")
		}
		set[id] = true
	}
}

func TestURLShortenerApp_shorten(t *testing.T) {
	app := URLShortenerApp{
		urlRepo: &InMemoryURLRepo{
			records: make([]InMemoryUrlRepoRecord, 0),
		},
		idGenerator: NewUniqueIDGenerator(),
	}

	shortUrl, err := app.shorten("www.google.com")
	if err != nil {
		t.Error(err)
	}

	if len(shortUrl) != shortURLLen {
		t.Error("The app should return a short URL of the correct length")
	}
}

func TestURLShortenerApp_redirect(t *testing.T) {
	longUrl := "www.google.com"
	app := URLShortenerApp{
		urlRepo: &InMemoryURLRepo{
			records: make([]InMemoryUrlRepoRecord, 1),
		},
		idGenerator: NewUniqueIDGenerator(),
	}

	shortUrl, err := app.shorten(longUrl)
	if err != nil {
		t.Error(err)
	}

	redirectUrl, err := app.redirect(shortUrl)
	if err != nil {
		t.Error(err)
	}

	if redirectUrl != longUrl {
		t.Error("The app should redirect to the correct long URL")
	}
}

func TestURLShortener_shorten_SameForSameLongURL(t *testing.T) {
	app := URLShortenerApp{
		urlRepo: &InMemoryURLRepo{
			records: make([]InMemoryUrlRepoRecord, 0),
		},
		idGenerator: NewUniqueIDGenerator(),
	}

	longUrl := "www.google.com"
	shortUrl1, err := app.shorten(longUrl)
	if err != nil {
		t.Error(err)
	}

	shortUrl2, err := app.shorten(longUrl)
	if err != nil {
		t.Error(err)
	}

	if shortUrl1 != shortUrl2 {
		t.Error("The app should not produce 2 different short URLs for the same long URL")
	}
}

func TestEncodeID_Zero(t *testing.T) {
	testBuf := [idByteLen]byte{}
	encodeID(&testBuf, 0, 0)
	for i := 0; i < idByteLen; i++ {
		if testBuf[i] != 0x00 {
			t.Error("Encoding 0 should result in all 0s")
		}
	}
}

func TestEncodeID_Simple(t *testing.T) {
	testBuf := [idByteLen]byte{}
	err := encodeID(&testBuf, 1, 1)
	if err != nil {
		t.Error(err)
	}
	if testBuf[0] != 0x00 {
		t.Error("Wrong ID outputted per spec encoding")
	}
	if testBuf[1] != 0x00 {
		t.Error("Wrong ID outputted per spec encoding")
	}
	if testBuf[2] != 0x00 {
		t.Error("Wrong ID outputted per spec encoding")
	}
	if testBuf[3] != 0x01 {
		t.Error("Wrong ID outputted per spec encoding")
	}
	if testBuf[4] != 0x00 {
		t.Error("Wrong ID outputted per spec encoding")
	}
	if testBuf[5] != 0x00 {
		t.Error("Wrong ID outputted per spec encoding")
	}
	if testBuf[6] != 0x01 {
		t.Error("Wrong ID outputted per spec encoding")
	}
}

func TestEncodeID_Random(t *testing.T) {
	testBuf := [idByteLen]byte{}
	err := encodeID(&testBuf, 0x12345678, 0b000100100011010001010110)
	if err != nil {
		t.Error(err)
	}
	if testBuf[0] != 0x12 {
		t.Error("Wrong ID outputted per spec encoding")
	}
	if testBuf[1] != 0x34 {
		t.Error("Wrong ID outputted per spec encoding")
	}
	if testBuf[2] != 0x56 {
		t.Error("Wrong ID outputted per spec encoding")
	}
	if testBuf[3] != 0x78 {
		t.Error("Wrong ID outputted per spec encoding")
	}
	if testBuf[4] != 0b00010010 {
		t.Error("Wrong ID outputted per spec encoding")
	}
	if testBuf[5] != 0b00110100 {
		t.Error("Wrong ID outputted per spec encoding")
	}
	if testBuf[6] != 0b01010110 {
		t.Error("Wrong ID outputted per spec encoding")
	}
}

func TestEncodeID_MaxSeq(t *testing.T) {
	testBuf := [idByteLen]byte{}
	maxSeq := uint32(math.Pow(2, 23) - 1)
	err := encodeID(&testBuf, 0, maxSeq)
	if err != nil {
		t.Error(err)
	}
}

func TestEncodeID_TooLargeSeq(t *testing.T) {
	testBuf := [idByteLen]byte{}
	seqOverflow := uint32(math.Pow(2, 23))
	err := encodeID(&testBuf, 0, seqOverflow)
	if err == nil {
		t.Error("Should have errored on too large seq")
	}
}

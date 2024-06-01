package main

import "testing"

func TestInMemoryURLRepo_StoreURLRecord(t *testing.T) {
	imur := InMemoryURLRepo{records: []InMemoryUrlRepoRecord{}}
	id := [idByteLen]byte{1, 2, 3, 4, 5}
	long := "long"
	short := "short"
	err := imur.StoreURLRecord(id, long, short)
	if err != nil {
		t.Error("StoreURLRecord should not return an error")
	}
	if len(imur.records) != 1 {
		t.Error("StoreURLRecord should add a record to the repo")
	}
	if imur.records[0].id != id || imur.records[0].longUrl != long || imur.records[0].shortUrl != short {
		t.Error("StoreURLRecord should store the record correctly")
	}
}

func TestInMemoryURLRepo_StoreURLRecord_Multiple(t *testing.T) {
	imur := InMemoryURLRepo{records: []InMemoryUrlRepoRecord{}}
	id1 := [idByteLen]byte{1, 2, 3, 4, 5}
	id2 := [idByteLen]byte{5, 4, 3, 2, 1}
	err := imur.StoreURLRecord(id1, "long1", "short1")
	if err != nil {
		t.Error("StoreURLRecord should not return an error")
	}
	err = imur.StoreURLRecord(id2, "long2", "short2")
	if err != nil {
		t.Error("StoreURLRecord should not return an error")
	}
	if len(imur.records) != 2 {
		t.Error("StoreURLRecord should add a record to the repo")
	}
	if imur.records[0].id != id1 || imur.records[0].longUrl != "long1" || imur.records[0].shortUrl != "short1" {
		t.Error("StoreURLRecord should store the record correctly")
	}
	if imur.records[1].id != id2 || imur.records[1].longUrl != "long2" || imur.records[1].shortUrl != "short2" {
		t.Error("StoreURLRecord should store the record correctly")
	}
}

func TestInMemoryURLRepo_GetShortURL(t *testing.T) {
	imur := InMemoryURLRepo{records: []InMemoryUrlRepoRecord{{[idByteLen]byte{1, 2, 3, 4, 5}, "long", "short"}}}
	short, err := imur.GetShortURL("long")
	if err != nil {
		t.Error("GetShortURL should not return an error")
	}
	if short != "short" {
		t.Error("GetShortURL should return the correct short URL")
	}
}

func TestInMemoryURLRepo_GetShortURL_MultipleRecords(t *testing.T) {
	imur := InMemoryURLRepo{records: []InMemoryUrlRepoRecord{{[idByteLen]byte{1, 2, 3, 4, 5}, "long1", "short1"}, {[idByteLen]byte{5, 4, 3, 2, 1}, "long2", "short2"}}}
	short, err := imur.GetShortURL("long2")
	if err != nil {
		t.Error("GetShortURL should not return an error")
	}
	if short != "short2" {
		t.Error("GetShortURL should return the correct short URL")
	}
}

func TestInMemoryURLRepo_GetLongURL(t *testing.T) {
	imur := InMemoryURLRepo{records: []InMemoryUrlRepoRecord{{[idByteLen]byte{1, 2, 3, 4, 5}, "long", "short"}}}
	long, err := imur.GetLongURL("short")
	if err != nil {
		t.Error("GetLongURL should not return an error")
	}
	if long != "long" {
		t.Error("GetLongURL should return the correct long URL")
	}
}

func TestInMemoryURLRepo_GetLongURL_MultipleRecords(t *testing.T) {
	imur := InMemoryURLRepo{records: []InMemoryUrlRepoRecord{{[idByteLen]byte{1, 2, 3, 4, 5}, "long1", "short1"}, {[idByteLen]byte{5, 4, 3, 2, 1}, "long2", "short2"}}}
	long, err := imur.GetLongURL("short2")
	if err != nil {
		t.Error("GetLongURL should not return an error")
	}
	if long != "long2" {
		t.Error("GetLongURL should return the correct long URL")
	}
}

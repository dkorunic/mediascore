// @license
// Copyright (C) 2018  Dinko Korunic
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/asdine/storm"
)

const cacheFolder = "MediaScore"
const cacheNameMovie = "movie.db"
const cacheNameTv = "tv.db"
const cachePerm = 0700

var userCacheDir string

// CacheEntry holds Movie/TV media information, Id hash (which is SHA256 hash of (Title,Year) for Movie or
// (Title,Year,Season,Episode)), basename hash etc.
type CacheEntry struct {
	Id           []byte `storm:"id"` // Movie:Title+Year; TV: Title+Year+Season+Episode
	BaseNameHash []byte `storm:"index"`
	Title        string
	Year         string
	EpisodeTitle string
	Season       string
	EpisodeNr    string
	ImdbRating   string
	RtRating     string
	McRating     string
	IsTv         bool
}

// openCache creates storm/bbolt cache databases for Movie/TV media and required folders either by using
// USER_CACHE_DIR environment variable or using system-specific UserCacheDir()
func openCache() (*storm.DB, *storm.DB, error) {
	if userCacheDir == "" {
		dir, err := os.UserCacheDir()
		if err != nil {
			return nil, nil, err
		}

		userCacheDir = dir
	}

	subDir := userCacheDir + string(os.PathSeparator) + cacheFolder
	err := os.MkdirAll(subDir, cachePerm)
	if err != nil {
		return nil, nil, err
	}

	dbMovie, err := storm.Open(subDir + string(os.PathSeparator) + cacheNameMovie)
	if err != nil {
		return nil, nil, err
	}

	dbTv, err := storm.Open(subDir + string(os.PathSeparator) + cacheNameTv)
	if err != nil {
		return dbMovie, nil, err
	}

	return dbMovie, dbTv, nil
}

// closeCache closes Movie/TV cache (storm/bbolt database) if it is initialized
func closeCache(db *storm.DB) error {
	if db == nil {
		return fmt.Errorf("cache not successfully initialized")
	}

	return db.Close()
}

// getCacheKey creates SHA256 hash out of any number of concatenated string slices
func getCacheKey(vars ...string) []byte {
	var buf bytes.Buffer
	for _, v := range vars {
		buf.WriteString(v)
	}

	h := sha256.New()
	h.Write(buf.Bytes())

	return h.Sum(nil)
}

// updateCache saves renderTable data into Movie/TV cache (storm/bbolt database)
func updateCache(db *storm.DB, v renderTable) error {
	if db == nil {
		return fmt.Errorf("cache not successfully initialized")
	}

	return db.Save(&v.data)
}

// getCacheOne returns a single cache entry matching value with fieldName contents in the cache (storm/bbolt database)
func getCacheOne(db *storm.DB, fieldName string, value interface{}, to interface{}) error {
	if db == nil {
		return fmt.Errorf("cache not successfully initialized")
	}

	return db.One(fieldName, value, to)
}

// cleanCache deletes all cache databases
func cleanCache() error {
	if userCacheDir == "" {
		dir, err := os.UserCacheDir()
		if err != nil {
			return err
		}

		userCacheDir = dir
	}

	subDir := userCacheDir + string(os.PathSeparator) + cacheFolder
	return os.RemoveAll(subDir)
}

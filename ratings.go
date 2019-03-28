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
	"github.com/dkorunic/gomdb"
	log "github.com/sirupsen/logrus"
)

var omdbKey string // OMDb API key: https://www.omdbapi.com/

// getRatings gathers OMDb, IMDB, RottenTomatoes and MediaCritic information about given media title with optional year,
// season and episode information and fully populated information structure is sent to rendering channel
func getRatings(baseName, mediaTitle string, mediaYear, mediaSeason, mediaEpisode int,
	channel chan<- renderTable) error {
	// Initial cache lookup with filename hash: we are not sure at this point if this is TV series of Movie, so lookup
	// in both cache tables
	var cacheEntry CacheEntry
	baseNameHash := getCacheKey(baseName)
	err := getCacheOne(cacheTv, "BaseNameHash", baseNameHash, &cacheEntry)
	if err != nil {
		log.Debugf("TV series file %v (decoded: %v/%v/%v/%v) not found in cache: %v", baseName, mediaTitle,
			mediaYear, mediaSeason, mediaEpisode, err)
	} else {
		channel <- renderTable{isCached: true, data: cacheEntry}
		return nil
	}
	err = getCacheOne(cacheMovie, "BaseNameHash", baseNameHash, &cacheEntry)
	if err != nil {
		log.Debugf("Movie file %v (decoded: %v/%v) not found in cache: %v", baseName, mediaTitle, mediaYear,
			err)
	} else {
		channel <- renderTable{isCached: true, data: cacheEntry}
		return nil
	}

	// Prepare OMDb query
	api := gomdb.Init(omdbKey)
	query := &gomdb.QueryData{Title: mediaTitle, Year: zString(mediaYear)}

	isTv := false
	if mediaSeason > 0 && mediaEpisode > 0 {
		isTv = true

		query.Season = zString(mediaSeason)
		query.Episode = zString(mediaEpisode)
		query.SearchType = gomdb.EpisodeSearch

	} else {
		query.SearchType = gomdb.MovieSearch
	}

	// OMDb query by title (type "t")
	res, err := api.MovieByTitle(query)
	if err != nil {
		log.Debugf("Could not find media %q in OMDb, will retry with IMDB lookup: %v", mediaTitle, err)

		// IMDB query by title
		imdbID, err := getImdbId(mediaTitle, mediaYear)
		if err != nil {
			log.Debugf("Could not query IMDB with media %q: %v", mediaTitle, err)
			return err
		}

		// do another OMDb query by IMDB Id (type "i")
		query.ImdbId = imdbID
		res, err = api.MovieByImdbID(query)
		if err != nil {
			log.Debugf("Could not query IMDB with for media %q and IMDB ID %v: %v", mediaTitle, query.ImdbId,
				err)
			return err
		}
	}

	// We have title, year, season and episode details and attempt to lookup them in cache
	if isTv {
		// hash(Title, Year, Season, Episode)
		keyId := getCacheKey(mediaTitle, res.Year, query.Season, query.Episode)
		err := getCacheOne(cacheTv, "Id", keyId, &cacheEntry)
		if err != nil {
			log.Debugf("TV series %v/%v/%v/%v (internal: %v) not found in cache: %v", mediaTitle, query.Year,
				query.Season, query.Episode, keyId, err)
		} else {
			channel <- renderTable{isCached: true, data: cacheEntry}
			return nil
		}
	} else {
		// hash(Title, Year)
		keyId := getCacheKey(mediaTitle, query.Year)
		err := getCacheOne(cacheMovie, "Id", keyId, &cacheEntry)
		if err != nil {
			log.Debugf("Movie %v/%v (internal: %v) not found in cache: %v", mediaTitle, mediaYear, keyId, err)
		} else {
			channel <- renderTable{isCached: true, data: cacheEntry}
			return nil
		}
	}

	// RottenTomatoes scraping: only if OMDb doesn't have RT score
	if res.TomatoRating == "N/A" {
		rt, err := getRtScore(mediaTitle, res.Title, mediaSeason, res.TomatoURL, isTv)
		if err != nil {
			log.Debugf("Could not get RottenTomatoes rating for media %q: %v", mediaTitle, err)
		} else {
			res.TomatoRating = rt
		}
	}

	// Metacritic scraping
	metaCriticRating, err := getMcScore(mediaTitle, res.Title, mediaYear, mediaSeason, mediaEpisode, isTv)
	if err != nil {
		log.Debugf("Could not get Metacritic rating for media %q: %v", mediaTitle, err)
		metaCriticRating = "N/A"
	}

	// We now have all data, send it to rendering and set cache flag to yes
	if isTv {
		cacheEntry = CacheEntry{Title: mediaTitle, Year: res.Year, EpisodeTitle: res.Title,
			Season: query.Season, EpisodeNr: query.Episode, ImdbRating: res.ImdbRating,
			RtRating: res.TomatoRating, McRating: metaCriticRating, BaseNameHash: getCacheKey(baseName),
			IsTv: isTv, Id: getCacheKey(mediaTitle, res.Year, query.Season, query.Episode)}
	} else {
		cacheEntry = CacheEntry{Title: mediaTitle, Year: res.Year, ImdbRating: res.ImdbRating,
			RtRating: res.TomatoRating, McRating: metaCriticRating, BaseNameHash: getCacheKey(baseName),
			IsTv: isTv, Id: getCacheKey(mediaTitle, res.Year)}
	}
	channel <- renderTable{isCached: false, data: cacheEntry}

	return nil
}

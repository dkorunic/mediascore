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
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
)

const mcBaseUrl = "https://www.metacritic.com"
const mcRefUrl = "http://www.metacritic.com/advanced-search" // MC referer
const mcResultLink = ".result a[href]"                       // search page result link
const mcMetaScoreSelector = ".phead_summary .metascore_w"    // MC Metascore
const mcUserScoreSelector = ".metascore_w.user"              // MC Userscore

// getMcScore initiates MetaScore search for media, gets MetaScore and if it is not available yet gets UserScore
func getMcScore(mediaTitle, omdbTitle string, mediaYear, mediaSeason, mediaEpisode int, isTv bool) (string, error) {
	// Always generate MC URL as OMDb doesn't provide it
	var mcUrl string
	if isTv {
		mcUrl = mcBaseUrl + "/search/tv/" + mediaTitle + getMcYearRange(mediaYear)
	} else {
		mcUrl = mcBaseUrl + "/search/movie/" + omdbTitle + getMcYearRange(mediaYear)
	}

	res, doc, err := getMediaDoc(mcUrl, mcRefUrl)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// Find first MC media page link in search results
	str, ok := doc.Find(mcResultLink).First().Attr("href")
	if !ok {
		log.Debugf("Unable to find Metacritic movie page, used search query: %v", mcUrl)
		return "N/A", nil
	}
	mcUrl = mcBaseUrl + str
	// If it's TV then append relevant season
	if isTv {
		mcUrl += getMcSeason(mediaSeason)
	}

	res2, doc, err := getMediaDoc(mcUrl, mcRefUrl)
	if err != nil {
		return "", err
	}
	defer res2.Body.Close()

	// Find MetaScore rating
	rating := doc.Find(mcMetaScoreSelector).First().Text()
	if rating == "" || !isInt(rating) {
		// If we don't have MetaScore, extract UserScore
		userRating := doc.Find(mcUserScoreSelector).First().Text()
		if len(userRating) > 0 {
			f, err := strconv.ParseFloat(userRating, 32)
			if err != nil {
				return "N/A", nil
			}

			// At least emulate the same range as MetaScore
			userRating = fmt.Sprintf("%d", int(f*10))

			return userRating, nil
		}

		log.Debugf("Unable to find Metacritic score (metascore or userscore), used media page: %v", mcUrl)
		return "N/A", nil
	}

	return rating, nil
}

// getMcYearRange creates a MC search filter with +1 year from a given date
func getMcYearRange(mediaYear int) string {
	return fmt.Sprintf("/results?date_range_from=01-01-%d&date_range_to=30-12-%d&search_type=advanced",
		mediaYear, mediaYear+1)
}

// getMcSeason creates a season suffix for MC TV media page
func getMcSeason(mediaSeason int) string {
	return fmt.Sprintf("/season-%d", mediaSeason)
}

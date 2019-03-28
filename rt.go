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
	"strings"
)

const rtBaseUrl = "https://www.rottentomatoes.com"
const rtTvScoreSelector = ".superPageFontColor.meter-align"                                             // RT TV score
const rtMovieScoreSelector = "span.mop-ratings-wrap__percentage.mop-ratings-wrap__percentage--audience" // RT Movie score

// getRtScore gets RottenTomatoes score
func getRtScore(mediaTitle, omdbTitle string, mediaSeason int, tomatoUrl string, isTv bool) (string, error) {
	// Generate RT media URL if OMDb doesn't provide it
	if tomatoUrl == "" || tomatoUrl == "N/A" {
		if isTv {
			tomatoUrl = rtBaseUrl + "/tv/" + getRtName(mediaTitle) + fmt.Sprintf("/s%0d", mediaSeason)
		} else {
			tomatoUrl = rtBaseUrl + "/m/" + getRtName(omdbTitle)
		}
	}

	res, doc, err := getMediaDoc(tomatoUrl, rtBaseUrl)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// Extract RT rating
	var rating string
	if isTv {
		rating = doc.Find(rtTvScoreSelector).
			First().Text()
	} else {
		rating = doc.Find(rtMovieScoreSelector).
			First().Text()
	}
	// Cleanup newlines, get first number, cleanup percentage signs
	rating = strings.Trim(strings.Trim(strings.Split(strings.TrimSpace(rating), " ")[0], "\n"), "%")

	if rating == "" || !isInt(rating) {
		return "N/A", nil
	}

	return rating, nil
}

// getRtName creates a RT-compatible media title, encoding spaces with underscore and removing colons
func getRtName(mediaTitle string) string {
	return strings.ReplaceAll(strings.ReplaceAll(mediaTitle, " ", "_"), ":", "")
}

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
	"net/http"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const defaultHTTPTimeout = 6 * time.Second // HTTP timeout at 6s

type renderTable struct {
	isCached bool
	data     CacheEntry
}

// getMediaDoc for a given URL does a HTTP GET and returns ready goquery document
func getMediaDoc(url string, refUrl string) (*http.Response, *goquery.Document, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Referer", refUrl)

	client := &http.Client{Timeout: defaultHTTPTimeout}
	res, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if res.StatusCode != 200 {
		return nil, nil, fmt.Errorf("HTTP error %v for URL: %v", res.StatusCode, req.URL.String())
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, nil, err
	}
	return res, doc, nil
}

// zString returns integer converted to string except for zero, which results in an empty string
func zString(input int) string {
	if input == 0 {
		return ""
	}
	return strconv.Itoa(input)
}

// absInt returns absolute value of integer
func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// isInt returns true if it is possible to convert string into string without any error and false otherwise
func isInt(v string) bool {
	if _, err := strconv.Atoi(v); err != nil {
		return false
	}
	return true
}

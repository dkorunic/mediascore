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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/asdine/storm"
	"github.com/tj/go-spin"

	log "github.com/sirupsen/logrus"

	"github.com/olekukonko/tablewriter"

	"github.com/middelink/go-parse-torrent-name"

	"github.com/karrick/godirwalk"
	"github.com/pborman/getopt"
)

const defaultPathnameQueueSize = 128                // store up to 128 path names to score
const defaultSpinningDelay = time.Millisecond * 200 // delay between spinner animations

var helpFlag, cleanFlag *bool
var videoExtensions map[string]int
var tableTvHeader, tableMovieHeader []string
var cacheMovie, cacheTv *storm.DB

func init() {
	helpFlag = getopt.BoolLong("help", 'h', "display help")
	cleanFlag = getopt.BoolLong("clean", 'c', "clean cache before scoring media")

	// Permitted video extensions
	videoExtensions = map[string]int{".3g2": 1, ".3gp": 1, ".3gp2": 1, ".asf": 1, ".avi": 1, ".divx": 1, ".flv": 1,
		".mk3d": 1, ".m4v": 1, ".mk2": 1, ".mka": 1, ".mkv": 1, ".mov": 1, ".mp4": 1, ".mp4a": 1, ".mpeg": 1, ".mpg": 1,
		".ogg": 1, ".ogm": 1, ".ogv": 1, ".qt": 1, ".ra": 1, ".ram": 1, ".rm": 1, ".ts": 1, ".wav": 1, ".webm": 1,
		".wma": 1, ".wmv": 1, ".iso": 1, ".vob": 1}

	// TV/Movie headers in rendered tables
	tableTvHeader = []string{"Title", "Year", "Episode Title", "Season", "Episode Nr", "IMDB rating", "RT rating",
		"Metacritic rating"}
	tableMovieHeader = []string{"Title", "Year", "IMDB rating", "RT rating", "Metacritic rating"}

	// Recognized env variables
	omdbKey = os.Getenv("OMDB_API_KEY")
	userCacheDir = os.Getenv("USER_CACHE_DIR")
	_, ok := os.LookupEnv("DEBUG")
	if ok {
		log.SetLevel(log.DebugLevel)
	}
}

func main() {
	// Getopt parameter/argument parser
	getopt.Parse()
	args := getopt.Args()

	// Show usage
	if *helpFlag || len(args) < 1 {
		getopt.PrintUsage(os.Stderr)
		os.Exit(0)
	}

	// Require OMDb key: limit is 1k queries per day for a free tier
	// Get yours here and/or donate: https://www.omdbapi.com/
	if omdbKey == "" {
		log.Error("Missing OMDb key. Please set OMDB_API_KEY environment variable.")
		os.Exit(1)
	}

	// Root context
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Attempt to clean cache
	if *cleanFlag {
		log.Info("Cleaned cache folder, continuing.")
		_ = cleanCache()
	}

	// Cache initialisation
	var err error
	cacheMovie, cacheTv, err = openCache()
	if err != nil {
		log.Debugf("Unable to open/create cache: %v", err)
	}
	defer closeCache(cacheTv)
	defer closeCache(cacheMovie)

	var wg sync.WaitGroup

	// Spinning wheel
	spinnerChan := make(chan struct{})

	if !log.IsLevelEnabled(log.DebugLevel) {
		// Spinning wheel
		wg.Add(1)
		go func(channel <-chan struct{}) {
			defer wg.Done()

			s := spin.New()
			s.Set(spin.Box3)
			tickerChan := time.NewTicker(defaultSpinningDelay)
			defer tickerChan.Stop()

			for {
				select {
				case _, ok := <-channel:
					if !ok {
						fmt.Printf("Done!\n")
						return
					}
				case <-tickerChan.C:
					fmt.Printf("\rChecking media: %s ", s.Next())
				case <-ctx.Done():
					return
				}
			}
		}(spinnerChan)
	}

	// Ingress rendering channel
	renderChan := make(chan renderTable, defaultPathnameQueueSize)

	// Output renderer
	wg.Add(1)
	go func(channel <-chan renderTable) {
		defer wg.Done()

		// Initialize TV and Movie table headers and style
		var tvTableCtr, movieTableCtr int
		tvTable := tvTableInit()
		movieTable := movieTableInit()

		for {
			select {
			case v, ok := <-channel:
				// Start rendering when channel has been closed
				if !ok {
					// Render Movie table only if not empty
					if movieTableCtr > 0 {
						movieTable.Render()
						if tvTableCtr > 0 {
							fmt.Print("\n")
						}
					}
					// Similarly render TV table only if not empty
					if tvTableCtr > 0 {
						tvTable.Render()
					}
					return
				}

				// Reformat and push to appropriate table, caching only if needed
				if v.data.IsTv {
					data := []string{v.data.Title, v.data.Year, v.data.EpisodeTitle, v.data.Season, v.data.EpisodeNr,
						v.data.ImdbRating, v.data.RtRating, v.data.McRating}
					tvTable.Append(data)
					tvTableCtr++

					if !v.isCached {
						_ = updateCache(cacheTv, v)
					}
				} else {
					data := []string{v.data.Title, v.data.Year, v.data.ImdbRating, v.data.RtRating, v.data.McRating}
					movieTable.Append(data)
					movieTableCtr++

					if !v.isCached {
						_ = updateCache(cacheMovie, v)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}(renderChan)

	// Process directories sequentially
	for i := range args {
		processDirectory(ctx, filepath.Clean(args[i]), renderChan)
	}

	close(renderChan)
	close(spinnerChan)
	wg.Wait()
}

// processDirectory processes each media folder, gets ranking data and sends it to rendering channel
func processDirectory(ctx context.Context, rootPath string, renderChan chan renderTable) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Signal handler variables
	signalChan := make(chan os.Signal, 1)

	// Signal handler: handle SIGUSR1, SIGUSR2 and SIGTERM and exit
	registerSignal(signalChan)
	go func(channel <-chan os.Signal) {
		for {
			select {
			case <-channel:
				fmt.Printf("\n")
				log.Warn("Exiting program as requested.")
				cancel()
				os.Exit(1)
			case <-ctx.Done():
				return
			}
		}
	}(signalChan)

	// Worker pool of media scoring routines
	var wg sync.WaitGroup
	fileChan := make(chan string, defaultPathnameQueueSize)
	for w := 0; w < runtime.NumCPU(); w++ {
		wg.Add(1)
		go func(channel <-chan string) {
			defer wg.Done()

			for {
				select {
				case v, ok := <-channel:
					if !ok {
						return
					}

					// Get all rankings and push to render channel
					getMovieInfo(v, renderChan)
				case <-ctx.Done():
					return
				}
			}
		}(fileChan)
	}

	// Fast concurrent directory walker: won't follow symlinks and won't sort entries
	err := godirwalk.Walk(rootPath, &godirwalk.Options{
		Unsorted:            true,
		FollowSymbolicLinks: false,
		// Default callback processes only directory entries
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			// Process only if entry is filename
			if de.IsRegular() {
				_, err := os.Stat(osPathname)
				if err != nil {
					return err
				}

				fileChan <- osPathname
			}
			return nil
		},
		// Default error callback skips over when encountering errors
		ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
			return godirwalk.SkipNode
		},
	})

	if err != nil {
		log.Errorf("Fatal directory walking error: %v", err)
		os.Exit(1)
	}

	// Close channels and cleanup routines
	close(fileChan)
	wg.Wait()
}

// getMovieInfo gets base name, checks if suffix is in recognized media suffixes, parses media information from the
// filename and gets ratings
func getMovieInfo(fullPath string, channel chan<- renderTable) {
	baseName := filepath.Base(fullPath)
	ext := filepath.Ext(baseName)

	if _, ok := videoExtensions[ext]; ok {
		info, err := parsetorrentname.Parse(baseName)
		if err != nil {
			log.Errorf("Not able to parse: %v", baseName)
		}

		// Strip parsetorrentname() results from creeping trailing/leading dots
		movieTitle := strings.Trim(info.Title, ".")
		err = getRatings(baseName, movieTitle, info.Year, info.Season, info.Episode, channel)
		if err != nil {
			log.Debug("Unable to get ratings for %v: %v", baseName, err)
		}
	}
}

// tvTableInit initializes TV table with header, formatting style, separator and borders
func tvTableInit() *tablewriter.Table {
	tvTable := tablewriter.NewWriter(os.Stdout)
	tvTable.SetHeader(tableTvHeader)
	tvTable.SetCaption(true, "TV Series Ratings ----------^")
	tvTable.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	tvTable.SetCenterSeparator("|")

	return tvTable
}

// movieTableInit initializes Movie table with header, formatting style, separator and borders
func movieTableInit() *tablewriter.Table {
	movieTable := tablewriter.NewWriter(os.Stdout)
	movieTable.SetHeader(tableMovieHeader)
	movieTable.SetCaption(true, "Movie Ratings ----------^")
	movieTable.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	movieTable.SetCenterSeparator("|")

	return movieTable
}

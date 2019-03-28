mediascore
===

[![GitHub license](https://img.shields.io/github/license/dkorunic/mediascore.svg)](https://github.com/dkorunic/mediascore/blob/master/LICENSE.txt)
[![GitHub release](https://img.shields.io/github/release/dkorunic/mediascore.svg)](https://github.com/dkorunic/mediascore/releases/latest)

## About

**Mediascore** is a CLI tool to display rankings for TV series or movies from IMDB, Rotten Tomatoes and Metacritic in a simple table format so that you can easily decide what to watch next and if it is worth your time.

Program will accept one or more paths being given and will parse all media files in attempt to classify media title, year, season and episode information and query [OMDb](https://www.omdbapi.com/), [IMDB](https://www.imdb.com/), [Rotten Tomatoes](https://www.rottentomatoes.com/) and [Mediacritic](https://www.metacritic.com/) searching for media information and rankings, eventually displaying separately the movie ranking table and the TV episode ranking table.

[![asciicast](https://asciinema.org/a/237341.svg)](https://asciinema.org/a/237341)

## Caveats

Using **mediascore** requires [OMDb](https://www.omdbapi.com/) API key, obtainable [here](https://www.omdbapi.com/apikey.aspx). Note that a free key is limited to 1,000 API calls per day, but all successful results are being cached. I strongly urge you to donate to Brian if you find OMDb useful.

Trying to score a large volume of media files could potentially lead to Rotten Tomatoes or Metacritic blacklisting your IP, so use wisely.

Media parsing information from the filename is being done by [parse-torrent-name](https://github.com/middelink/go-parse-torrent-name) Go library which is not without issues, so make sure to have [Kodi-compatible file naming](https://kodi.wiki/view/Naming_video_files) if possible.

Rotten Tomatoes and Mediacritic scraping is simply horribly done, I know.

## Installation

### Manual

Download your preferred flavor from [the releases](https://github.com/dkorunic/mediascore/releases/latest) page and install manually.

### Using go get

```shell
go get https://github.com/dkorunic/mediascore
```

## Usage

```shell
Usage: mediascore [-ch] [parameters ...]
 -c, --clean  clean cache before scoring media
 -h, --help   display help
```

Typical use case is to invoke **mediascore** on one or more media (for instance ones exported through SMB to Kodi or Plex) network/local folders like below:

```shell
OMDB_API_KEY=XXX ./mediascore "/Volumes/XBMC/TV Shows"
```

We are exporting OMDb API key as environment variable `OMDB_API_KEY` and using mediascore to parse locally mounted XMBC volume. Environment variable `OMDB_API_KEY` can also be permanently set and exported in your shell profile/configuration files for future use.

When unsure what is **mediascore** doing, you can also set `DEBUG=1` environment variable for a bit more verbosity.

## Bugs, feature requests, etc.

Please open a PR or report an issue. Thanks!
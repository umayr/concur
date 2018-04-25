package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/umayr/sync"
)

var (
	flagSubreddit    string
	flagPages        int
	flagRedirectURI  string
	flagPlaylistID   string
	flagRefreshToken string
)

func init() {
	flag.StringVar(
		&flagPlaylistID,
		"playlist-id",
		"",
		"playlist id where tracks are going to be added",
	)
	flag.StringVar(
		&flagRefreshToken,
		"refresh-token",
		"",
		"refresh token to create a new authentication token",
	)
	flag.StringVar(
		&flagSubreddit,
		"subreddit",
		"music",
		"comma separated subreddit names",
	)
	flag.IntVar(
		&flagPages,
		"pages",
		3,
		"max pages to be parsed",
	)
	flag.StringVar(
		&flagRedirectURI,
		"redirect-url",
		"http://localhost:8080/callback",
		"redirect URI registered in spotify application",
	)

	flag.Parse()
}

func exitWithErr(err error) {
	fmt.Println(err.Error())
	os.Exit(1)
}

func newSpotifyWithRedirectURI() *sync.Spotify {
	cb, err := url.Parse(flagRedirectURI)
	if err != nil {
		exitWithErr(err)
	}

	s, err := sync.NewSpotify(cb.String(), "")
	if err != nil {
		exitWithErr(err)
	}

	var route string
	if strings.HasPrefix(cb.Path, "/") {
		route = cb.Path
	} else {
		route = "/" + cb.Path
	}

	http.HandleFunc(route, s.Callback)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "This server is for handling callback request from Spotify authentication.")
	})

	var port string
	if cb.Port() == "" {
		port = "80"
	} else {
		port = cb.Port()
	}
	go http.ListenAndServe(fmt.Sprintf(":%s", port), nil)

	fmt.Printf("Please log in to Spotify by visiting the following page in your browser: %s\n", s.AuthURL())
	time.AfterFunc(time.Minute, func() {
		if !s.Ready() {
			fmt.Println("Unable to get authorised by Spotify")
			os.Exit(1)
		}
	})
	<-s.Done
	return s
}

func newSpotifyWithRefreshToken() *sync.Spotify {
	s, err := sync.NewSpotifyWithRefreshToken(flagRefreshToken)
	if err != nil {
		exitWithErr(err)
	}

	return s
}

func main() {
	var s *sync.Spotify
	if flagRefreshToken == "" {
		s = newSpotifyWithRedirectURI()
	} else {
		s = newSpotifyWithRefreshToken()
	}

	var subs []string
	if strings.Contains(flagSubreddit, ",") {
		subs = append(strings.Split(flagSubreddit, ","))
	} else {
		subs = append(subs, flagSubreddit)
	}

	var list []string
	for _, v := range subs {
		r := &sync.Reddit{Max: flagPages, Subreddit: v}
		l, err := r.Fetch()
		if err != nil {
			exitWithErr(err)
		}

		list = append(list, l...)
	}

	idx, err := s.Search(list)
	if err != nil {
		exitWithErr(err)
	}

	if flagPlaylistID == "" {
		name := "Reddit Sync - "
		for _, str := range strings.Split(flagSubreddit, ",") {
			name += "/r/" + str + " "
		}
		p, err := s.CreatePlaylist(name)
		if err != nil {
			exitWithErr(err)
		}

		flagPlaylistID = p
	}

	c, err := s.AddToPlaylist(flagPlaylistID, idx...)
	if err != nil {
		exitWithErr(err)
	}

	fmt.Printf("Added (%d/%d) tracks in the playlist.", c, len(idx))
}

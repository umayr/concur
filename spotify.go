package sync

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

var (
	ErrNotReady = fmt.Errorf("client is not ready, make sure user has authenticated the app")
	ErrNoEnv    = fmt.Errorf("client key and secret is not set in environment variables")

	scopes = []string{spotify.ScopePlaylistModifyPublic, spotify.ScopeUserReadPrivate}
)

type Spotify struct {
	Done chan bool

	auth   spotify.Authenticator
	client spotify.Client
	state  string
	ready  bool
}

const (
	ClientKey    = "SPOTIFY_ID"
	ClientSecret = "SPOTIFY_SECRET"
	AuthURL      = "https://accounts.spotify.com/authorize"
	TokenURL     = "https://accounts.spotify.com/api/token"
)

func NewSpotify(redirectURI, state string) (*Spotify, error) {
	if os.Getenv(ClientKey) == "" && os.Getenv(ClientSecret) == "" {
		return nil, ErrNoEnv
	}

	s := &Spotify{
		auth:  spotify.NewAuthenticator(redirectURI, scopes...),
		Done:  make(chan bool),
		state: state,
	}

	return s, nil
}

func NewSpotifyWithRefreshToken(refreshToken string) (*Spotify, error) {
	logf("requesting new access token with provided refresh token")
	if os.Getenv(ClientKey) == "" && os.Getenv(ClientSecret) == "" {
		return nil, ErrNoEnv
	}

	s := &Spotify{
		auth:  spotify.NewAuthenticator(""),
		Done:  make(chan bool),
		state: "",
	}

	t := oauth2.Token{
		RefreshToken: refreshToken,
	}

	cfg := &oauth2.Config{
		ClientID:     os.Getenv(ClientKey),
		ClientSecret: os.Getenv(ClientSecret),
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  AuthURL,
			TokenURL: TokenURL,
		},
	}

	tr := &http.Transport{
		TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
	}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Transport: tr})

	ts := cfg.TokenSource(ctx, &t)
	token, err := ts.Token()
	if err != nil {
		return nil, err
	}

	s.client = s.auth.NewClient(token)
	close(s.Done)
	s.ready = true

	return s, nil
}

func (s *Spotify) Callback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	logf("callback invoked with code: %s and state: %s", q.Get("code"), q.Get("state"))

	t, err := s.auth.Token(s.state, r)
	if err != nil {
		logf("unable to get token from Spotify: %s", err)
		http.Error(w, "Couldn't get token from Spotify", http.StatusForbidden)
		return
	}

	buf, err := json.Marshal(t)
	if err != nil {
		logf("unable to marshal token: %s", err)
	}

	logf("oauth2 token: %s", string(buf))

	if sf := r.FormValue("state"); sf != s.state {
		logf("state value mismatched (%s != %s)", sf, s.state)
		http.NotFound(w, r)
		return
	}

	logf("creating client with token")
	s.client = s.auth.NewClient(t)
	fmt.Fprint(w, "login process completed.")
	s.ready = true
	s.Done <- s.ready
}

func (s *Spotify) AuthURL() string {
	return s.auth.AuthURL(s.state)
}

func (s *Spotify) Ready() bool {
	return s.ready
}

func (s *Spotify) CurrentUser() (string, error) {
	if !s.ready {
		return "", ErrNotReady
	}
	user, err := s.client.CurrentUser()
	if err != nil {
		logf("unable to get current user")
		return "", err
	}

	return string(user.ID), err
}

func (s *Spotify) Search(query []string) ([]string, error) {
	if !s.ready {
		return nil, ErrNotReady
	}

	var (
		limit  = 1
		offset = 0
		opts   = spotify.Options{Limit: &limit, Offset: &offset}
	)
	var idx []string
	for _, t := range query {
		q :=
			strings.TrimSpace(
				strings.Replace(
					// cleaning up query as majorly people use brackets to define genre
					regexp.
						MustCompile(`[\[(].*[])]`).
						ReplaceAllString(t, ""),
					"--",
					"-",
					-1),
			)
		logf("searching for %s", q)

		results, err := s.client.SearchOpt(q, spotify.SearchTypeTrack, &opts)
		if err != nil {
			logf("error occurred while searching for %s: %s", q, err)
			return nil, err
		}

		if results.Tracks != nil && len(results.Tracks.Tracks) > 0 {
			logf("found tracks for %s, picking first track", q)
			idx = append(idx, results.Tracks.Tracks[0].ID.String())
		}
	}

	logf("found %d tracks for %d queries", len(idx), len(query))
	return idx, nil
}

func breakIDs(s []spotify.ID, limit int) [][]spotify.ID {
	l := float64(len(s))
	f := math.Ceil(l / float64(limit))
	r := int(f + math.Copysign(0.5, f))

	x := make([][]spotify.ID, r)
	for i := 0; i < r; i++ {
		b, e := i*limit, i*limit+limit
		if e > len(s) {
			e = len(s)
		}
		x[i] = s[b:e]
	}

	return x
}

func copyTracks(dict map[string]string, src []spotify.PlaylistTrack) {
	for _, v := range src {
		dict[v.Track.ID.String()] = v.Track.Name
	}
}

func (s *Spotify) CreatePlaylist(name string) (string, error) {
	if !s.ready {
		return "", ErrNotReady
	}
	userID, err := s.CurrentUser()
	if err != nil {
		return "", err
	}

	p, err := s.client.CreatePlaylistForUser(userID, name, true)
	if err != nil {
		return "", err
	}

	return p.ID.String(), nil
}

func (s *Spotify) AddToPlaylist(playlistID string, idx ...string) (int, error) {
	if !s.ready {
		return 0, ErrNotReady
	}

	userID, err := s.CurrentUser()
	if err != nil {
		return 0, err
	}

	p, err := s.client.GetPlaylist(userID, spotify.ID(playlistID))
	if err != nil {
		return 0, err
	}

	dict := make(map[string]string)
	copyTracks(dict, p.Tracks.Tracks)

	limit := 100
	f := math.Ceil(float64(p.Tracks.Total) / float64(limit))
	iter := int(f + math.Copysign(0.5, f))

	for i := 1; i < iter; i++ {
		offset := i * limit
		p, err := s.client.GetPlaylistTracksOpt(
			userID,
			spotify.ID(playlistID),
			&spotify.Options{Limit: &limit, Offset: &offset},
			"items(track(name,id))",
		)
		if err != nil {
			return 0, err
		}

		copyTracks(dict, p.Tracks)
	}

	var list []spotify.ID
	for _, id := range idx {
		if _, exists := dict[id]; !exists {
			list = append(list, spotify.ID(id))
		} else {
			logf(`track "%s" already in the playlist "%s"`, dict[id], p.Name)
		}
	}

	for _, c := range breakIDs(list, 100) {
		if _, err := s.client.AddTracksToPlaylist(userID, spotify.ID(playlistID), c...); err != nil {
			return 0, err
		}
	}

	return len(list), nil
}
